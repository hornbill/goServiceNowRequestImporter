package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"github.com/hornbill/color"
	_ "github.com/hornbill/go-mssqldb" //Microsoft SQL Server driver - v2005+
	"github.com/hornbill/goapiLib"
	_ "github.com/hornbill/mysql"    //MySQL v4.1 to v5.x and MariaDB driver
	_ "github.com/hornbill/mysql320" //MySQL v3.2.0 to v5 driver
	"github.com/hornbill/pb"
	"github.com/hornbill/spinner"
	"github.com/hornbill/sqlx"
	"html"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	version           = "1.0"
	appServiceManager = "com.hornbill.servicemanager"
)

var (
	appDBDriver            string
	arrCallsLogged         = make(map[string]reqRelStruct)
	arrCallDetailsMaps     = make([]map[string]interface{}, 0)
	arrActivityDetailsMaps = make([]map[string]interface{}, 0)
	boolConfLoaded         bool
	configFileName         string
	configZone             string
	configDryRun           bool
	configDebug            bool
	configMaxRoutines      string
	connStrAppDB           string
	counters               counterTypeStruct
	mapGenericConf         snCallConfStruct
	mapActivityConf        snActivityConfStruct
	analysts               []analystListStruct
	categories             []categoryListStruct
	closeCategories        []categoryListStruct
	customers              []customerListStruct
	priorities             []priorityListStruct
	services               []serviceListStruct
	sites                  []siteListStruct
	teams                  []teamListStruct
	snImportConf           snImportConfStruct
	timeNow                string
	startTime              time.Time
	endTime                time.Duration
	espXmlmc               *apiLib.XmlmcInstStruct
	xmlmcInstanceConfig    xmlmcConfigStruct
	mutexAnalysts          = &sync.Mutex{}
	mutexArrCallsLogged    = &sync.Mutex{}
	mutexBar               = &sync.Mutex{}
	mutexCategories        = &sync.Mutex{}
	mutexCloseCategories   = &sync.Mutex{}
	mutexCustomers         = &sync.Mutex{}
	mutexLogging           = &sync.Mutex{}
	mutexPriorities        = &sync.Mutex{}
	mutexServices          = &sync.Mutex{}
	mutexSites             = &sync.Mutex{}
	mutexTeams             = &sync.Mutex{}
	wgRequest              sync.WaitGroup
	wgAssoc                sync.WaitGroup
	wgAttach               sync.WaitGroup
	reqPrefix              string
	maxGoroutines          = 1
	boolProcessAttachments bool
)

// ----- Structures -----
type counterTypeStruct struct {
	sync.Mutex
	created        int
	createdSkipped int
	filesAttached  int
}

//----- Config Data Structs
type snImportConfStruct struct {
	HBConf                    hbConfStruct //Hornbill Instance connection details
	CustomerType              string
	SNAppDBConf               appDBConfStruct //ServiceNow Database connection details
	ConfIncident              snCallConfStruct
	ConfServiceRequest        snCallConfStruct
	ConfChangeRequest         snCallConfStruct
	ConfProblem               snCallConfStruct
	ConfKnownError            snCallConfStruct
	ConfActivities            snActivityConfStruct
	TeamMapping               map[string]interface{}
	CategoryMapping           map[string]interface{}
	ResolutionCategoryMapping map[string]interface{}
}
type hbConfStruct struct {
	UserName   string
	Password   string
	InstanceID string
	URL        string
}
type appDBConfStruct struct {
	Driver   string
	Server   string
	Database string
	UserName string
	Password string
	Port     int
	Encrypt  bool
}
type snCallConfStruct struct {
	Import                 bool
	CallClass              string
	DefaultTeam            string
	DefaultPriority        string
	DefaultService         string
	SQLStatement           map[string]interface{}
	CoreFieldMapping       map[string]interface{}
	AdditionalFieldMapping map[string]interface{}
	StatusMapping          map[string]interface{}
	PriorityMapping        map[string]interface{}
	ServiceMapping         map[string]interface{}
}

type snActivityConfStruct struct {
	Import       bool
	SQLStatement map[string]interface{}
	Category     string
	ParentRef    string
	Title        string
	Description  string
	StartDate    string
	DueDate      string
	AssignTo     string
	Status       string
	Decision     string
	Reason       string
}

//----- XMLMC Config and Interaction Structs
type xmlmcConfigStruct struct {
	instance string
	url      string
	zone     string
}
type xmlmcResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
	TaskID       string      `xml:"taskId"`
}

//----- Shared Structs -----
type stateStruct struct {
	Code     string `xml:"code"`
	ErrorRet string `xml:"error"`
}

//----- Data Structs -----

type xmlmcSysSettingResponse struct {
	MethodResult string      `xml:"status,attr"`
	State        stateStruct `xml:"state"`
	Setting      string      `xml:"params>option>value"`
}

//----- Request Logged Structs
type xmlmcRequestResponseStruct struct {
	MethodResult string      `xml:"status,attr"`
	RequestID    string      `xml:"params>primaryEntityData>record>h_pk_reference"`
	SiteCountry  string      `xml:"params>rowData>row>h_country"`
	State        stateStruct `xml:"state"`
}
type xmlmcBPMSpawnedStruct struct {
	MethodResult string      `xml:"status,attr"`
	Identifier   string      `xml:"params>identifier"`
	State        stateStruct `xml:"state"`
}

//----- Site Structs
type siteListStruct struct {
	SiteName string
	SiteID   int
}
type xmlmcSiteListResponse struct {
	MethodResult string      `xml:"status,attr"`
	SiteID       int         `xml:"params>rowData>row>h_id"`
	SiteName     string      `xml:"params>rowData>row>h_site_name"`
	SiteCountry  string      `xml:"params>rowData>row>h_country"`
	State        stateStruct `xml:"state"`
}

//----- Priority Structs
type priorityListStruct struct {
	PriorityName string
	PriorityID   int
}
type xmlmcPriorityListResponse struct {
	MethodResult string      `xml:"status,attr"`
	PriorityID   int         `xml:"params>rowData>row>h_pk_priorityid"`
	PriorityName string      `xml:"params>rowData>row>h_priorityname"`
	State        stateStruct `xml:"state"`
}

//----- Service Structs
type serviceListStruct struct {
	ServiceName          string
	ServiceID            int
	ServiceBPMIncident   string
	ServiceBPMService    string
	ServiceBPMChange     string
	ServiceBPMProblem    string
	ServiceBPMKnownError string
}
type xmlmcServiceListResponse struct {
	MethodResult  string      `xml:"status,attr"`
	ServiceID     int         `xml:"params>rowData>row>h_pk_serviceid"`
	ServiceName   string      `xml:"params>rowData>row>h_servicename"`
	BPMIncident   string      `xml:"params>rowData>row>h_incident_bpm_name"`
	BPMService    string      `xml:"params>rowData>row>h_service_bpm_name"`
	BPMChange     string      `xml:"params>rowData>row>h_change_bpm_name"`
	BPMProblem    string      `xml:"params>rowData>row>h_problem_bpm_name"`
	BPMKnownError string      `xml:"params>rowData>row>h_knownerror_bpm_name"`
	State         stateStruct `xml:"state"`
}

//----- Team Structs
type teamListStruct struct {
	TeamName string
	TeamID   string
}
type xmlmcTeamListResponse struct {
	MethodResult string      `xml:"status,attr"`
	TeamID       string      `xml:"params>rowData>row>h_id"`
	TeamName     string      `xml:"params>rowData>row>h_name"`
	State        stateStruct `xml:"state"`
}

//----- Category Structs
type categoryListStruct struct {
	CategoryCode string
	CategoryID   string
	CategoryName string
}
type xmlmcCategoryListResponse struct {
	MethodResult string      `xml:"status,attr"`
	CategoryID   string      `xml:"params>id"`
	CategoryName string      `xml:"params>fullname"`
	State        stateStruct `xml:"state"`
}

//----- Audit Structs
type xmlmcAuditListResponse struct {
	MethodResult     string      `xml:"status,attr"`
	TotalStorage     float64     `xml:"params>maxStorageAvailble"`
	TotalStorageUsed float64     `xml:"params>totalStorageUsed"`
	State            stateStruct `xml:"state"`
}

//----- Analyst Structs
type analystListStruct struct {
	AnalystID   string
	AnalystName string
}
type xmlmcAnalystListResponse struct {
	MethodResult     string      `xml:"status,attr"`
	AnalystFullName  string      `xml:"params>name"`
	AnalystFirstName string      `xml:"params>firstName"`
	AnalystLastName  string      `xml:"params>lastName"`
	State            stateStruct `xml:"state"`
}

//----- Customer Structs
type customerListStruct struct {
	CustomerID   string
	CustomerName string
}
type xmlmcCustomerListResponse struct {
	MethodResult      string      `xml:"status,attr"`
	CustomerFirstName string      `xml:"params>firstName"`
	CustomerLastName  string      `xml:"params>lastName"`
	State             stateStruct `xml:"state"`
}

//----- Associated Record Struct
type reqRelStruct struct {
	SNRequestGUID string
	SNParentRef   string
	SMCallRef     string
}

//----- File Attachment Structs
type xmlmcAttachmentResponse struct {
	MethodResult    string      `xml:"status,attr"`
	ContentLocation string      `xml:"params>contentLocation"`
	State           stateStruct `xml:"state"`
}

//----- File Attachment Struct
type fileAssocStruct struct {
	ContentType string  `db:"content_type"`
	FileGUID    string  `db:"sys_id"`
	SizeU       float64 `db:"size_bytes"`
	SizeC       float64 `db:"size_compressed"`
	FileName    string  `db:"file_name"`
	AddedBy     string  `db:"sys_created_by"`
	TimeAdded   string  `db:"sys_created_on"`
	Pieces      int     `db:"pieces"`
	FileDataB64 string
	SMCallRef   string
}

//----- File Attachment Data Struct
type fileAssocDataStruct struct {
	Position int    `db:"position"`
	Length   int    `db:"length"`
	Data     string `db:"data"`
}

// main package
func main() {
	//-- Start Time for Durration
	startTime = time.Now()
	//-- Start Time for Log File
	timeNow = time.Now().Format(time.RFC3339)
	timeNow = strings.Replace(timeNow, ":", "-", -1)

	//-- Grab and Parse Flags
	flag.StringVar(&configFileName, "file", "conf.json", "Name of the configuration file to load")
	flag.StringVar(&configZone, "zone", "eur", "Override the default Zone the instance sits in")
	flag.BoolVar(&configDryRun, "dryrun", false, "Dump import XML to log instead of creating requests")
	flag.BoolVar(&configDebug, "debug", false, "Full DEBUG output to log file")
	flag.StringVar(&configMaxRoutines, "concurrent", "1", "Maximum number of requests to import concurrently.")
	flag.BoolVar(&boolProcessAttachments, "attachments", true, "Defaults to true. Set to false to skip the import of file attachments.")
	flag.Parse()

	//-- Output to CLI and Log
	logger(1, "---- ServiceNow Task Import Utility V"+fmt.Sprintf("%v", version)+" ----", true)
	logger(1, "Flag - Config File "+fmt.Sprintf("%s", configFileName), true)
	logger(1, "Flag - Zone "+fmt.Sprintf("%s", configZone), true)
	logger(1, "Flag - Dry Run "+fmt.Sprintf("%v", configDryRun), true)
	logger(1, "Flag - Debug Logger "+fmt.Sprintf("%v", configDebug), true)
	logger(1, "Flag - Concurrent Requests "+fmt.Sprintf("%v", configMaxRoutines), true)
	logger(1, "Flag - Import Attachments "+fmt.Sprintf("%v", boolProcessAttachments), true)

	//Check maxGoroutines for valid value
	maxRoutines, err := strconv.Atoi(configMaxRoutines)
	if err != nil {
		color.Red("Unable to convert maximum concurrency of [" + configMaxRoutines + "] to type INT for processing")
		return
	}
	maxGoroutines = maxRoutines

	if maxGoroutines < 1 || maxGoroutines > 10 {
		color.Red("The maximum concurrent requests allowed is between 1 and 10 (inclusive).\n\n")
		color.Red("You have selected " + configMaxRoutines + ". Please try again, with a valid value against ")
		color.Red("the -concurrent switch.")
		return
	}

	//-- Load Configuration File Into Struct
	snImportConf, boolConfLoaded = loadConfig()
	if boolConfLoaded != true {
		logger(4, "Unable to load config, process closing.", true)
		return
	}

	//Set SQL driver ID string for Application Data
	if snImportConf.SNAppDBConf.Driver == "" {
		logger(4, "Database Driver not set in configuration.", true)
		return
	}

	if snImportConf.SNAppDBConf.Driver == "mysql" || snImportConf.SNAppDBConf.Driver == "mssql" || snImportConf.SNAppDBConf.Driver == "mysql320" {
		appDBDriver = snImportConf.SNAppDBConf.Driver
	} else {
		logger(4, "The SQL driver ("+snImportConf.SNAppDBConf.Driver+") for the ServiceNow Application Database specified in the configuration file is not valid.", true)
		return
	}

	//-- Set Instance ID
	SetInstance(configZone, snImportConf.HBConf.InstanceID)
	//-- Generate Instance XMLMC Endpoint
	snImportConf.HBConf.URL = getInstanceURL()

	//-- Log in to Hornbill instance
	var boolLogin = login()
	if boolLogin != true {
		logger(4, "Unable to Login ", true)
		return
	}
	//-- Defer log out of Hornbill instance until after main() is complete
	defer logout()

	//-- Build DB connection strings for ServiceNow Data Source
	connStrAppDB = buildConnectionString()

	//Process Incidents
	mapGenericConf = snImportConf.ConfIncident
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("IN")
		processCallData()
	}
	//Process Service Requests
	mapGenericConf = snImportConf.ConfServiceRequest
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("SR")
		processCallData()
	}
	//Process Change Requests
	mapGenericConf = snImportConf.ConfChangeRequest
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("CH")
		processCallData()
	}
	//Process Problems
	mapGenericConf = snImportConf.ConfProblem
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("PM")
		processCallData()
	}
	//Process Known Errors
	mapGenericConf = snImportConf.ConfKnownError
	if mapGenericConf.Import == true {
		reqPrefix = getRequestPrefix("KE")
		processCallData()
	}

	if len(arrCallsLogged) > 0 {
		//We have new calls logged
		//Now process File Attachments
		processRequestAttachments()

		//Now process activities
		mapActivityConf = snImportConf.ConfActivities
		if mapActivityConf.Import == true {
			processActivities()
		}
		//Now process associations
		processCallAssociations()
	}

	//-- End output
	logger(1, "Requests Logged: "+fmt.Sprintf("%d", counters.created), true)
	logger(1, "Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), true)
	logger(1, "Files Attached: "+fmt.Sprintf("%d", counters.filesAttached), true)
	//-- Show Time Takens
	endTime = time.Now().Sub(startTime)
	logger(1, "Time Taken: "+fmt.Sprintf("%v", endTime), true)
	logger(1, "---- ServiceNow Call Import Complete ---- ", true)
}

//getRequestPrefix - gets and returns current maxResultsAllowed sys setting value
func getRequestPrefix(callclass string) string {
	espXmlmc, sessErr := NewEspXmlmcSession()
	if sessErr != nil {
		logger(4, "Unable to attach to XMLMC session to get Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}
	strSetting := ""
	switch callclass {
	case "IN":
		strSetting = "guest.app.requests.types.IN"
	case "SR":
		strSetting = "guest.app.requests.types.SR"
	case "CH":
		strSetting = "app.requests.types.CH"
	case "PM":
		strSetting = "app.requests.types.PM"
	case "KE":
		strSetting = "app.requests.types.KE"
	}

	espXmlmc.SetParam("appName", appServiceManager)
	espXmlmc.SetParam("filter", strSetting)
	response, err := espXmlmc.Invoke("admin", "appOptionGet")
	if err != nil {
		logger(4, "Could not retrieve System Setting for Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}
	var xmlRespon xmlmcSysSettingResponse
	err = xml.Unmarshal([]byte(response), &xmlRespon)
	if err != nil {
		logger(4, "Could not retrieve System Setting for Request Prefix. Using default ["+callclass+"].", false)
		return callclass
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Could not retrieve System Setting for Request Prefix: "+xmlRespon.MethodResult, false)
		return callclass
	}
	return xmlRespon.Setting
}

//processRequestAttachments - process associations between requests
func processRequestAttachments() {
	time.Sleep(100 * time.Millisecond)
	intRequestsRaised := len(arrCallsLogged)
	strRequestsRaised := strconv.Itoa(intRequestsRaised)
	logger(1, "Processing File Attachments for "+strRequestsRaised+" imported requests. Please wait...", true)
	bar := pb.StartNew(intRequestsRaised)
	//Process each association record, insert in to Hornbill
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
	for requestID, requestSlice := range arrCallsLogged {

		snCallRef := requestID
		snCallGUID := requestSlice.SNRequestGUID
		smCallRef := requestSlice.SMCallRef

		maxGoroutinesGuard <- struct{}{}
		wgAttach.Add(1)
		go func() {
			defer wgAttach.Done()
			time.Sleep(1 * time.Millisecond)
			//We have Master and Slave calls matched in the SM database
			processFileAttachments(snCallGUID, snCallRef, smCallRef)

			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()
			<-maxGoroutinesGuard
		}()
	}
	wgAttach.Wait()
	bar.FinishPrint("Request Attachments Processing Complete")
	logger(1, "Request Attachments Processing Complete", false)
	return
}

func processFileAttachments(taskSysID, snCallRef, smCallRef string) {
	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connStrAppDB)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Request File Attachments: "+fmt.Sprintf("%v", err), false)
		return
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error for Request File Attachments: "+fmt.Sprintf("%v", err), false)
		return
	}
	//build query
	sqlFileQuery := "SELECT sys_id, file_name, content_type, size_bytes, size_compressed, sys_created_by, sys_created_on, "
	sqlFileQuery += " (SELECT COUNT(position) FROM sys_attachment_doc WHERE sys_attachment_doc.sys_attachment = sys_attachment.sys_id GROUP BY sys_attachment_doc.sys_attachment ) AS pieces "
	sqlFileQuery += " FROM sys_attachment WHERE table_sys_id = '" + taskSysID + "'"

	if configDebug {
		logger(1, "[DATABASE] Connection Successful for File Attachments", false)
		logger(1, "[DATABASE] Running query for Request File Attachments against ServiceNow ref ["+snCallRef+"] Service Manager ref ["+smCallRef+"]. Please wait...", false)
		logger(1, "[DATABASE] Request File Attachments Query: "+sqlFileQuery, false)
	}

	//Run Query
	attachmentRows, err := db.Queryx(sqlFileQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return
	}
	//-- Iterate through file attachment records returned from SQL query
	for attachmentRows.Next() {
		//Scan current file attachment record in to struct
		var requestAttachment fileAssocStruct
		err = attachmentRows.StructScan(&requestAttachment)
		if err != nil {
			logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", err), false)
			return
		}
		requestAttachment.SMCallRef = smCallRef

		//Now go get each of the file chunks for processing
		sqlFileDataQuery := "SELECT position, length, data "
		sqlFileDataQuery += " FROM sys_attachment_doc "
		sqlFileDataQuery += " WHERE sys_attachment = '" + requestAttachment.FileGUID + "'"
		sqlFileDataQuery += " ORDER BY position ASC"

		if configDebug {
			logger(1, "[DATABASE] Request File Attachments Query: "+sqlFileDataQuery, false)
		}

		//Run Query
		attachmentRowData, err := db.Queryx(sqlFileDataQuery)
		if err != nil {
			logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		}

		boolBreakRowLoop := false
		var attachSlice []byte

		//-- Iterate through file attachment records returned from SQL query:
		for attachmentRowData.Next() {
			var rowAttachmentData fileAssocDataStruct
			err = attachmentRowData.StructScan(&rowAttachmentData)
			if err != nil {
				logger(4, " Data Mapping Error: "+fmt.Sprintf("%v", err), false)
				boolBreakRowLoop = true
				break
			}
			//Decode Base64 string in to byte slice
			decoded, err := base64.StdEncoding.DecodeString(rowAttachmentData.Data)
			if err != nil {
				logger(4, " Error Decoding Base64: "+fmt.Sprintf("%v", err), false)
				boolBreakRowLoop = true
				break
			}
			attachSlice = append(attachSlice, decoded...)
		}
		if !boolBreakRowLoop {
			boolDecompressed := true
			//Attachment byte slice in to a reader
			attachReader := bytes.NewReader(attachSlice)
			//Attachment Reader in to a gzip reader
			gzipReader, err := gzip.NewReader(attachReader)
			if err != nil {
				logger(4, " Error creating gzip reader: "+fmt.Sprintf("%v", err), false)
				boolDecompressed = false
			}
			//Read gzip Reader (uncompressed data) in to byte slice using io.ReadAll
			unComSlice, readererr := ioutil.ReadAll(gzipReader)
			if readererr != nil {
				logger(4, " Error creating gzip reader: "+fmt.Sprintf("%v", readererr), false)
				boolDecompressed = false
			}
			if boolDecompressed {
				dataEncoded := base64.StdEncoding.EncodeToString(unComSlice)
				gzipReader.Close()
				requestAttachment.FileDataB64 = dataEncoded
				if !addFileAttachmentToRequest(requestAttachment) {
					//File attachment not added!
				}
			}
		}
	}
}

//addFileAttachmentToRequest - takes the fileRecord data, attach this to request and update content location
func addFileAttachmentToRequest(fileRecord fileAssocStruct) bool {
	attPriKey := fileRecord.SMCallRef
	useFileName := fileRecord.FileName
	filenameReplacer := strings.NewReplacer("<", "_", ">", "_", "|", "_", "\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_")
	useFileName = fmt.Sprintf("%s", filenameReplacer.Replace(useFileName))
	espXmlmc, sessErr2 := NewEspXmlmcSession()
	if sessErr2 != nil {
		logger(4, "Unable to attach to XMLMC session to add file record.", true)
		return false
	}
	//File content read - add data to instance
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Requests")
	espXmlmc.SetParam("keyValue", attPriKey)
	espXmlmc.SetParam("folder", "/")
	espXmlmc.OpenElement("localFile")
	espXmlmc.SetParam("fileName", useFileName)
	espXmlmc.SetParam("fileData", fileRecord.FileDataB64)
	espXmlmc.CloseElement("localFile")
	espXmlmc.SetParam("overwrite", "true")
	var XMLSTRINGDATA = espXmlmc.GetParam()
	XMLAttach, xmlmcErr := espXmlmc.Invoke("data", "entityAttachFile")
	if xmlmcErr != nil {
		logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+attPriKey+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
		return false
	}
	var xmlRespon xmlmcAttachmentResponse

	err := xml.Unmarshal([]byte(XMLAttach), &xmlRespon)
	if err != nil {
		logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+attPriKey+"]: "+fmt.Sprintf("%v", err), false)
		logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Could not add Attachment File Data for ["+useFileName+"] ["+attPriKey+"]: "+xmlRespon.State.ErrorRet, false)
			logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
		} else {
			//-- If we've got a Content Location back from the API, update the file record with this
			if xmlRespon.ContentLocation != "" {
				strService := ""
				strMethod := ""
				espXmlmc, sessErr3 := NewEspXmlmcSession()
				if sessErr3 != nil {
					logger(4, "Unable to attach to XMLMC session to add file record.", true)
					return false
				}
				espXmlmc.SetParam("application", appServiceManager)
				espXmlmc.SetParam("entity", "RequestAttachments")
				espXmlmc.OpenElement("primaryEntityData")
				espXmlmc.OpenElement("record")
				espXmlmc.SetParam("h_request_id", attPriKey)
				espXmlmc.SetParam("h_description", "Originally added by "+fileRecord.AddedBy)
				espXmlmc.SetParam("h_filename", useFileName)
				espXmlmc.SetParam("h_contentlocation", xmlRespon.ContentLocation)
				espXmlmc.SetParam("h_timestamp", fileRecord.TimeAdded)
				espXmlmc.SetParam("h_visibility", "trustedGuest")
				espXmlmc.CloseElement("record")
				espXmlmc.CloseElement("primaryEntityData")
				strService = "data"
				strMethod = "entityAddRecord"

				XMLSTRINGDATA = espXmlmc.GetParam()
				XMLContentLoc, xmlmcErrContent := espXmlmc.Invoke(strService, strMethod)
				if xmlmcErrContent != nil {
					logger(4, "Could not update request ["+attPriKey+"] with attachment ["+useFileName+"]: "+fmt.Sprintf("%v", xmlmcErrContent), false)
					logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
					return false
				}
				var xmlResponLoc xmlmcResponse

				err := xml.Unmarshal([]byte(XMLContentLoc), &xmlResponLoc)
				if err != nil {
					logger(4, "Added file data to but unable to set Content Location on ["+attPriKey+"] for File Content ["+useFileName+"] - read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
					logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
					return false
				}
				if xmlResponLoc.MethodResult != "ok" {
					logger(4, "Added file data but unable to set Content Location on ["+attPriKey+"] for File Content ["+useFileName+"]: "+xmlResponLoc.State.ErrorRet, false)
					logger(1, "File Data Record XML "+fmt.Sprintf("%s", XMLSTRINGDATA), false)
					return false
				}
				counters.Lock()
				counters.filesAttached++
				counters.Unlock()
				logger(1, "File ["+useFileName+"] added to ["+attPriKey+"]", false)
			}
		}
	}
	return true
}

//confirmResponse - prompts user, expects a fuzzy yes or no response, does not continue until this is given
func confirmResponse() bool {
	var cmdResponse string
	_, errResponse := fmt.Scanln(&cmdResponse)
	if errResponse != nil {
		log.Fatal(errResponse)
	}
	if cmdResponse == "y" || cmdResponse == "yes" || cmdResponse == "Y" || cmdResponse == "Yes" || cmdResponse == "YES" {
		return true
	} else if cmdResponse == "n" || cmdResponse == "no" || cmdResponse == "N" || cmdResponse == "No" || cmdResponse == "NO" {
		return false
	} else {
		color.Red("Please enter yes or no to continue:")
		return confirmResponse()
	}
}

//processActivities - take records to insert as activities against Service Manager requests
func processActivities() {
	time.Sleep(100 * time.Millisecond)
	logger(1, "Processing Activities, please wait...", true)
	if queryDBCallDetails("Activity", connStrAppDB) == true {
		bar := pb.StartNew(len(arrActivityDetailsMaps))
		//We have Call Details - insert them in to
		maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
		for _, callRecord := range arrActivityDetailsMaps {
			maxGoroutinesGuard <- struct{}{}
			wgRequest.Add(1)
			callRecordArr := callRecord
			strParentRefMapping := mapActivityConf.ParentRef
			parentRef := getFieldValue(strParentRefMapping, callRecordArr)

			go func() {
				defer wgRequest.Done()
				time.Sleep(1 * time.Millisecond)
				mutexBar.Lock()
				bar.Increment()
				mutexBar.Unlock()
				smImported, impOk := arrCallsLogged[parentRef]
				smCallRef := smImported.SMCallRef
				if impOk && smCallRef != "" && smCallRef != "<nil>" {
					boolActivity := addActivity(callRecordArr, smCallRef)
					if boolActivity {
						logger(3, "[ACTIVITY] Activity raised against Service Manager request ["+smCallRef+"]", false)
					} else {
						logger(4, "Failed Raising Activity for SM Request ["+smCallRef+"]", false)
					}
				}
				<-maxGoroutinesGuard
			}()
		}
		wgRequest.Wait()

		bar.FinishPrint("Request Activity Import Complete")
	} else {
		logger(4, "Request Search Failed for Request Activities.", true)
	}
}

//addActivity - Adds an Activity against an imported Request
func addActivity(callMap map[string]interface{}, smCallRef string) bool {

	strTitle := getFieldValue(mapActivityConf.Title, callMap)
	strDescription := getFieldValue(mapActivityConf.Description, callMap)
	strCategory := getFieldValue(mapActivityConf.Category, callMap)
	strStartDate := getFieldValue(mapActivityConf.StartDate, callMap)
	strDueDate := getFieldValue(mapActivityConf.DueDate, callMap)
	strAssignTo := getFieldValue(mapActivityConf.AssignTo, callMap)
	//Is strStatus = closed, close the activity once it's been raised
	strStatus := getFieldValue(mapActivityConf.Status, callMap)
	strDecision := getFieldValue(mapActivityConf.Decision, callMap)
	strReason := getFieldValue(mapActivityConf.Reason, callMap)

	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("title", strTitle)
	if strDescription != "" {
		espXmlmc.SetParam("details", strDescription)
	}
	espXmlmc.SetParam("category", strCategory)
	if strStartDate != "" {
		espXmlmc.SetParam("startDate", strStartDate)
	}
	if strDueDate != "" {
		espXmlmc.SetParam("dueDate", strDueDate)
	}
	if strAssignTo != "" {
		boolUserExists := doesAnalystExist(strAssignTo)
		if !boolUserExists {
			boolUserExists = doesCustomerExist(strAssignTo)
		}
		if boolUserExists {
			espXmlmc.SetParam("assignTo", "urn:sys:user:"+strAssignTo)
		}
	}
	if strCategory == "BPM Authorisation" {
		espXmlmc.OpenElement("outcome")
		espXmlmc.SetParam("outcome", "accept")
		espXmlmc.OpenElement("displayName")
		espXmlmc.SetParam("text", "Authorise")
		espXmlmc.CloseElement("displayName")
		espXmlmc.SetParam("buttonColor", "default")
		espXmlmc.SetParam("requiresReason", "false")
		espXmlmc.CloseElement("outcome")
		espXmlmc.OpenElement("outcome")
		espXmlmc.SetParam("outcome", "refuse")
		espXmlmc.OpenElement("displayName")
		espXmlmc.SetParam("text", "Rejected")
		espXmlmc.CloseElement("displayName")
		espXmlmc.SetParam("buttonColor", "default")
		espXmlmc.SetParam("requiresReason", "true")
		espXmlmc.CloseElement("outcome")
	} else {
		espXmlmc.OpenElement("outcome")
		espXmlmc.SetParam("outcome", "done")
		espXmlmc.OpenElement("displayName")
		espXmlmc.SetParam("text", "Done")
		espXmlmc.CloseElement("displayName")
		espXmlmc.SetParam("buttonColor", "default")
		espXmlmc.SetParam("requiresReason", "false")
		espXmlmc.CloseElement("outcome")
	}
	espXmlmc.SetParam("objectRefUrn", "urn:sys:entity:com.hornbill.servicemanager:Requests:"+smCallRef)
	espXmlmc.SetParam("remindAssignee", "false")
	espXmlmc.SetParam("remindOwner", "false")

	//Debug - Output XML request to log
	if configDebug {
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Raise Activity XML "+fmt.Sprintf("%s", XMLSTRING), false)
	}
	//END Debug

	XMLCreate, xmlmcErr := espXmlmc.Invoke("task", "taskCreate2")

	if xmlmcErr != nil {
		logger(4, "Unable to create activity on Hornbill instance:"+fmt.Sprintf("%v", xmlmcErr), false)
		return false
	}
	var xmlRespon xmlmcResponse

	err = xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
	if err != nil {
		logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
		return false
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Unable to log request: "+xmlRespon.State.ErrorRet, false)
		return false
	}
	if xmlRespon.TaskID != "" && xmlRespon.TaskID != "<nil>" && (strStatus == "Closed Complete" || strStatus == "Closed Incomplete") {
		//Mark the task as COMPLETE
		espXmlmc, err := NewEspXmlmcSession()
		if err != nil {
			return false
		}
		espXmlmc.SetParam("taskId", xmlRespon.TaskID)
		espXmlmc.SetParam("outcome", strDecision+"\n"+strReason)
		XMLCreate, xmlmcErr := espXmlmc.Invoke("task", "taskComplete")

		if xmlmcErr != nil {
			logger(4, "Unable to complete activity on Hornbill instance: "+fmt.Sprintf("%v", xmlmcErr), false)
			return false
		}
		var xmlTaskRespon xmlmcResponse

		err = xml.Unmarshal([]byte(XMLCreate), &xmlTaskRespon)
		if err != nil {
			logger(4, "Unable to read complete activity response on Hornbill instance:"+fmt.Sprintf("%v", err), false)
			return false
		}
		if xmlTaskRespon.MethodResult != "ok" {
			logger(4, "Unable to complete activity: "+xmlTaskRespon.State.ErrorRet, false)
			return false
		}
	}
	return true
}

//processCallAssociations - process associations between requests
func processCallAssociations() {
	time.Sleep(100 * time.Millisecond)
	intRequestsRaised := len(arrCallsLogged)
	strRequestsRaised := strconv.Itoa(intRequestsRaised)
	logger(1, "Processing Request Associations for "+strRequestsRaised+" imported requests. Please wait...", true)
	bar := pb.StartNew(intRequestsRaised)
	//Process each association record, insert in to Hornbill
	maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
	for _, requestSlice := range arrCallsLogged {

		snParentRef := requestSlice.SNParentRef
		smCallRef := requestSlice.SMCallRef
		smMasterRef := arrCallsLogged[snParentRef].SMCallRef

		maxGoroutinesGuard <- struct{}{}
		wgAssoc.Add(1)
		go func() {
			defer wgAssoc.Done()
			time.Sleep(1 * time.Millisecond)
			if smMasterRef != "" && smMasterRef != "<nil>" && smCallRef != "" {
				//We have Master and Slave calls matched in the SM database
				addAssocRecord(smMasterRef, smCallRef)
			}
			mutexBar.Lock()
			bar.Increment()
			mutexBar.Unlock()
			<-maxGoroutinesGuard
		}()
	}
	wgAssoc.Wait()
	bar.FinishPrint("Request Association Processing Complete")
	logger(1, "Request Association Processing Complete", false)
}

//addAssocRecord - given a Master Reference and a Slave Refernce, adds a call association record to Service Manager
func addAssocRecord(masterRef, slaveRef string) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return
	}
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "RelatedRequests")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_fk_parentrequestid", masterRef)
	espXmlmc.SetParam("h_fk_childrequestid", slaveRef)
	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")
	XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
	if xmlmcErr != nil {
		//		log.Fatal(xmlmcErr)
		logger(4, "Unable to create Request Association between ["+masterRef+"] and ["+slaveRef+"] :"+fmt.Sprintf("%v", xmlmcErr), false)
		return
	}
	var xmlRespon xmlmcResponse
	errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
	if errXMLMC != nil {
		logger(4, "Unable to read response from Hornbill instance for Request Association between ["+masterRef+"] and ["+slaveRef+"] :"+fmt.Sprintf("%v", errXMLMC), false)
		return
	}
	if xmlRespon.MethodResult != "ok" {
		logger(3, "Unable to add Request Association between ["+masterRef+"] and ["+slaveRef+"] : "+xmlRespon.State.ErrorRet, false)
		return
	}
	if configDebug {
		logger(1, "Request Association Success between ["+masterRef+"] and ["+slaveRef+"]", false)
	}
}

//processCallData - Query ServiceNow call data, process accordingly
func processCallData() {
	if queryDBCallDetails(mapGenericConf.CallClass, connStrAppDB) == true {
		time.Sleep(100 * time.Millisecond)
		fmt.Println("")
		logger(1, "Importing records. Please wait...", true)
		bar := pb.StartNew(len(arrCallDetailsMaps))
		//We have Call Details - insert them in to
		maxGoroutinesGuard := make(chan struct{}, maxGoroutines)
		for _, callRecord := range arrCallDetailsMaps {
			maxGoroutinesGuard <- struct{}{}
			wgRequest.Add(1)
			callRecordArr := callRecord
			callRecordCallref := fmt.Sprintf("%s", callRecord["callref"])

			go func() {
				defer wgRequest.Done()
				time.Sleep(1 * time.Millisecond)
				mutexBar.Lock()
				bar.Increment()
				mutexBar.Unlock()
				boolCallLogged, hbCallRef := logNewCall(mapGenericConf.CallClass, callRecordArr, callRecordCallref)
				if boolCallLogged {
					logger(3, "[REQUEST] Request "+hbCallRef+" raised from Task "+callRecordCallref, false)
				} else {
					logger(4, mapGenericConf.CallClass+" request log failed: "+callRecordCallref, false)
				}
				<-maxGoroutinesGuard
			}()
		}
		wgRequest.Wait()

		bar.FinishPrint(mapGenericConf.CallClass + " Request Import Complete")
	} else {
		logger(4, "Request Search Failed for Request Class: "+mapGenericConf.CallClass, true)
	}
}

//queryDBCallDetails -- Query call data & set map of calls to add to Hornbill
func queryDBCallDetails(callClass, connString string) bool {
	if callClass == "" || connString == "" {
		return false
	}
	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connString)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	logger(3, "[DATABASE] Connection Successful", false)
	logger(3, "[DATABASE] Running query for tasks of class "+callClass+". Please wait...", false)

	spin := spinner.New(spinner.CharSets[35], 300*time.Millisecond) // Build a new spinner
	spin.Prefix = "Running query for tasks of class " + callClass + ". Please wait "
	spin.Start()
	defer spin.Stop()

	strSQLQuery := ""
	//build query
	if callClass == "Activity" {
		arrQueryLen := len(mapActivityConf.SQLStatement)
		for i := 0; i < arrQueryLen; i++ {
			strSQLQuery += " " + fmt.Sprintf("%s", mapActivityConf.SQLStatement[strconv.Itoa(i)])
		}
	} else {
		arrQueryLen := len(mapGenericConf.SQLStatement)
		for i := 0; i < arrQueryLen; i++ {
			strSQLQuery += " " + fmt.Sprintf("%s", mapGenericConf.SQLStatement[strconv.Itoa(i)])
		}
	}
	if configDebug {
		logger(1, "[DATABASE] Query to retrieve "+callClass+" tasks from ServiceNow data: "+strSQLQuery, false)
	}

	//Run Query
	rows, err := db.Queryx(strSQLQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), true)
		return false
	}
	if callClass == "Activity" {
		//Clear down existing Call Details map
		arrActivityDetailsMaps = nil
		//Build map full of calls to import
		for rows.Next() {
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			//Stick marshalled data map in to parent slice
			arrActivityDetailsMaps = append(arrActivityDetailsMaps, results)
		}

	} else {
		//Clear down existing Call Details map
		arrCallDetailsMaps = nil
		//Build map full of calls to import
		for rows.Next() {
			results := make(map[string]interface{})
			err = rows.MapScan(results)
			//Stick marshalled data map in to parent slice
			arrCallDetailsMaps = append(arrCallDetailsMaps, results)
		}
	}
	defer rows.Close()
	return true
}

//logNewCall - Function takes ServiceNow call data in a map, and logs to Hornbill
func logNewCall(callClass string, callMap map[string]interface{}, snCallID string) (bool, string) {

	boolCallLoggedOK := false
	strNewCallRef := ""

	strStatus := ""
	statusMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_status"])
	if statusMapping != "" {
		strStatus = fmt.Sprintf("%s", mapGenericConf.StatusMapping[getFieldValue(statusMapping, callMap)])
	}

	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, ""
	}

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Requests")
	espXmlmc.SetParam("returnModifiedData", "true")
	espXmlmc.OpenElement("primaryEntityData")
	espXmlmc.OpenElement("record")
	strAttribute := ""
	strMapping := ""
	strServiceBPM := ""
	boolUpdateLogDate := false
	strLoggedDate := ""
	strClosedDate := ""
	//Loop through core fields from config, add to XMLMC Params
	for k, v := range mapGenericConf.CoreFieldMapping {
		boolAutoProcess := true
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)

		//Owning Analyst Name
		if strAttribute == "h_ownerid" {
			strOwnerID := getFieldValue(strMapping, callMap)
			if strOwnerID != "" {
				boolAnalystExists := doesAnalystExist(strOwnerID)
				if boolAnalystExists {
					//Get analyst from cache as exists
					analystIsInCache, strOwnerName := recordInCache(strOwnerID, "Analyst")
					if analystIsInCache && strOwnerName != "" {
						espXmlmc.SetParam(strAttribute, strOwnerID)
						espXmlmc.SetParam("h_ownername", strOwnerName)
					}
				}
			}
			boolAutoProcess = false
		}

		//Customer ID & Name
		if strAttribute == "h_fk_user_id" {
			strCustID := getFieldValue(strMapping, callMap)
			if strCustID != "" {
				boolCustExists := doesCustomerExist(strCustID)
				if boolCustExists {
					//Get customer from cache as exists
					customerIsInCache, strCustName := recordInCache(strCustID, "Customer")
					if customerIsInCache && strCustName != "" {
						espXmlmc.SetParam(strAttribute, strCustID)
						espXmlmc.SetParam("h_fk_user_name", strCustName)
					}
				}
			}
			boolAutoProcess = false
		}

		//Priority ID & Name
		//-- Get Priority ID
		if strAttribute == "h_fk_priorityid" {
			strPriorityID := getFieldValue(strMapping, callMap)
			strPriorityMapped, strPriorityName := getCallPriorityID(strPriorityID)
			if strPriorityMapped == "" && mapGenericConf.DefaultPriority != "" {
				strPriorityID = getPriorityID(mapGenericConf.DefaultPriority)
				strPriorityName = mapGenericConf.DefaultPriority
			}
			espXmlmc.SetParam(strAttribute, strPriorityMapped)
			espXmlmc.SetParam("h_fk_priorityname", strPriorityName)
			boolAutoProcess = false
		}

		// Category ID & Name
		if strAttribute == "h_category_id" && strMapping != "" {
			//-- Get Call Category ID
			strCategoryID, strCategoryName := getCallCategoryID(callMap, "Request")
			if strCategoryID != "" && strCategoryName != "" {
				espXmlmc.SetParam(strAttribute, strCategoryID)
				espXmlmc.SetParam("h_category", strCategoryName)
			}
			boolAutoProcess = false
		}

		// Closure Category ID & Name
		if strAttribute == "h_closure_category_id" && strMapping != "" {
			strClosureCategoryID, strClosureCategoryName := getCallCategoryID(callMap, "Closure")
			if strClosureCategoryID != "" {
				espXmlmc.SetParam(strAttribute, strClosureCategoryID)
				espXmlmc.SetParam("h_closure_category", strClosureCategoryName)
			}
			boolAutoProcess = false
		}

		// Service ID & Name, & BPM Workflow
		if strAttribute == "h_fk_serviceid" {
			//-- Get Service ID
			snServiceID := getFieldValue(strMapping, callMap)
			strServiceID := getCallServiceID(snServiceID)
			if strServiceID == "" && mapGenericConf.DefaultService != "" {
				strServiceID = getServiceID(mapGenericConf.DefaultService)
			}
			if strServiceID != "" {
				//-- Get record from Service Cache
				strServiceName := ""
				mutexServices.Lock()
				for _, service := range services {
					if strconv.Itoa(service.ServiceID) == strServiceID {
						strServiceName = service.ServiceName
						switch callClass {
						case "Incident":
							strServiceBPM = service.ServiceBPMIncident
						case "Service Request":
							strServiceBPM = service.ServiceBPMService
						case "Change Request":
							strServiceBPM = service.ServiceBPMChange
						case "Problem":
							strServiceBPM = service.ServiceBPMProblem
						case "Known Error":
							strServiceBPM = service.ServiceBPMKnownError
						}
					}
				}
				mutexServices.Unlock()

				if strServiceName != "" {
					espXmlmc.SetParam(strAttribute, strServiceID)
					espXmlmc.SetParam("h_fk_servicename", strServiceName)
				}
			}
			boolAutoProcess = false
		}

		// Request Status
		if strAttribute == "h_status" {
			espXmlmc.SetParam(strAttribute, strStatus)
			boolAutoProcess = false
		}

		// Team ID and Name
		if strAttribute == "h_fk_team_id" {
			//-- Get Team ID
			snTeamID := getFieldValue(strMapping, callMap)
			strTeamID, strTeamName := getCallTeamID(snTeamID)
			if strTeamID == "" && mapGenericConf.DefaultTeam != "" {
				strTeamName = mapGenericConf.DefaultTeam
				strTeamID = getTeamID(strTeamName)
			}
			if strTeamID != "" && strTeamName != "" {
				espXmlmc.SetParam(strAttribute, strTeamID)
				espXmlmc.SetParam("h_fk_team_name", strTeamName)
			}
			boolAutoProcess = false
		}

		// Site ID and Name
		if strAttribute == "h_site_id" {
			//-- Get site ID
			siteID, siteName := getSiteID(callMap)
			if siteID != "" && siteName != "" {
				espXmlmc.SetParam(strAttribute, siteID)
				espXmlmc.SetParam("h_site", siteName)
			}
			boolAutoProcess = false
		}

		// Resolved Date/Time
		if strAttribute == "h_dateresolved" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed") {
			strResolvedDate := getFieldValue(strMapping, callMap)
			if strResolvedDate != "" {
				espXmlmc.SetParam(strAttribute, strResolvedDate)
			}
		}

		// Closed Date/Time
		if strAttribute == "h_dateclosed" && strMapping != "" && (strStatus == "status.resolved" || strStatus == "status.closed" || strStatus == "status.onHold") {
			strClosedDate = getFieldValue(strMapping, callMap)
			if strClosedDate != "" {
				espXmlmc.SetParam(strAttribute, strClosedDate)
			}
		}

		// Log Date/Time - setup ready to be processed after call logged
		if strAttribute == "h_datelogged" && strMapping != "" {
			strLoggedDate = getFieldValue(strMapping, callMap)
			if strLoggedDate != "" {
				boolUpdateLogDate = true
			}
		}

		//Everything Else
		if boolAutoProcess &&
			strAttribute != "h_requesttype" &&
			strAttribute != "h_request_prefix" &&
			strAttribute != "h_category" &&
			strAttribute != "h_closure_category" &&
			strAttribute != "h_fk_servicename" &&
			strAttribute != "h_fk_team_name" &&
			strAttribute != "h_site" &&
			strAttribute != "h_fk_priorityname" &&
			strAttribute != "h_ownername" &&
			strAttribute != "h_fk_user_name" &&
			strAttribute != "h_datelogged" &&
			strAttribute != "h_dateresolved" &&
			strAttribute != "h_dateclosed" {

			if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
				espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
			}
		}

	}

	//Add request class & prefix
	espXmlmc.SetParam("h_requesttype", callClass)
	espXmlmc.SetParam("h_request_prefix", reqPrefix)

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("primaryEntityData")

	//Class Specific Data Insert
	espXmlmc.OpenElement("relatedEntityData")
	espXmlmc.SetParam("relationshipName", "Call Type")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	strAttribute = ""
	strMapping = ""
	//Loop through AdditionalFieldMapping fields from config, add to XMLMC Params if not empty
	for k, v := range mapGenericConf.AdditionalFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strMapping = fmt.Sprintf("%v", v)
		if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
			espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//Extended Data Insert
	espXmlmc.OpenElement("relatedEntityData")
	espXmlmc.SetParam("relationshipName", "Extended Information")
	espXmlmc.SetParam("entityAction", "insert")
	espXmlmc.OpenElement("record")
	espXmlmc.SetParam("h_request_type", callClass)
	strAttribute = ""
	strMapping = ""
	//Loop through AdditionalFieldMapping fields from config, add to XMLMC Params if not empty
	for k, v := range mapGenericConf.AdditionalFieldMapping {
		strAttribute = fmt.Sprintf("%v", k)
		strSubString := "h_custom_"
		if strings.Contains(strAttribute, strSubString) {
			strAttribute = convExtendedColName(strAttribute)
			strMapping = fmt.Sprintf("%v", v)
			if strMapping != "" && getFieldValue(strMapping, callMap) != "" {
				espXmlmc.SetParam(strAttribute, getFieldValue(strMapping, callMap))
			}
		}
	}

	espXmlmc.CloseElement("record")
	espXmlmc.CloseElement("relatedEntityData")

	//-- Check for Dry Run
	if configDryRun != true {

		XMLCreate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
		if xmlmcErr != nil {
			//log.Fatal(xmlmcErr)
			logger(4, "Unable to log request on Hornbill instance:"+fmt.Sprintf("%v", xmlmcErr), false)
			return false, "No"
		}
		var xmlRespon xmlmcRequestResponseStruct

		err := xml.Unmarshal([]byte(XMLCreate), &xmlRespon)
		if err != nil {
			counters.Lock()
			counters.createdSkipped++
			counters.Unlock()
			logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", err), false)
			return false, "No"
		}
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to log request: "+xmlRespon.State.ErrorRet, false)
			counters.Lock()
			counters.createdSkipped++
			counters.Unlock()
		} else {
			strNewCallRef = xmlRespon.RequestID

			var requestRelate reqRelStruct
			requestRelate.SMCallRef = strNewCallRef
			requestRelate.SNParentRef = fmt.Sprintf("%+s", callMap["parent_task_ref"])
			requestRelate.SNRequestGUID = fmt.Sprintf("%+s", callMap["request_guid"])
			mutexArrCallsLogged.Lock()
			arrCallsLogged[snCallID] = requestRelate
			mutexArrCallsLogged.Unlock()

			counters.Lock()
			counters.created++
			counters.Unlock()
			boolCallLoggedOK = true

			//Now update the request to create the activity stream
			espXmlmc.SetParam("socialObjectRef", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
			espXmlmc.SetParam("content", "Request imported from ServiceNow")
			espXmlmc.SetParam("visibility", "public")
			espXmlmc.SetParam("type", "Logged")
			fixed, err := espXmlmc.Invoke("activity", "postMessage")
			if err != nil {
				logger(5, "Activity Stream Creation failed for Request: "+strNewCallRef, false)
			} else {
				var xmlRespon xmlmcResponse
				err = xml.Unmarshal([]byte(fixed), &xmlRespon)
				if err != nil {
					logger(5, "Activity Stream Creation unmarshall failed for Request "+strNewCallRef, false)
				} else {
					if xmlRespon.MethodResult != "ok" {
						logger(5, "Activity Stream Creation was unsuccessful for ["+strNewCallRef+"]: "+xmlRespon.MethodResult, false)
					} else {
						if configDebug {
							logger(1, "Activity Stream Creation successful for ["+strNewCallRef+"]", false)
						}
					}
				}
			}

			//Now update Logdate
			if boolUpdateLogDate {
				espXmlmc.SetParam("application", appServiceManager)
				espXmlmc.SetParam("entity", "Requests")
				espXmlmc.OpenElement("primaryEntityData")
				espXmlmc.OpenElement("record")
				espXmlmc.SetParam("h_pk_reference", strNewCallRef)
				espXmlmc.SetParam("h_datelogged", strLoggedDate)
				espXmlmc.CloseElement("record")
				espXmlmc.CloseElement("primaryEntityData")
				XMLBPM, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
				if xmlmcErr != nil {
					//log.Fatal(xmlmcErr)
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse

				errLogDate := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
				if errLogDate != nil {
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+fmt.Sprintf("%v", errLogDate), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(4, "Unable to update Log Date of request ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet, false)
				}
			}

			//Now do BPM Processing
			if strStatus != "status.resolved" &&
				strStatus != "status.closed" &&
				strStatus != "status.cancelled" {
				if configDebug {
					logger(1, callClass+" Logged: "+strNewCallRef+". Open Request status, spawing BPM Process "+strServiceBPM, false)
				}
				if strNewCallRef != "" && strServiceBPM != "" {
					espXmlmc.SetParam("application", appServiceManager)
					espXmlmc.SetParam("name", strServiceBPM)
					espXmlmc.OpenElement("inputParams")
					espXmlmc.SetParam("objectRefUrn", "urn:sys:entity:"+appServiceManager+":Requests:"+strNewCallRef)
					espXmlmc.SetParam("requestId", strNewCallRef)
					espXmlmc.CloseElement("inputParams")

					XMLBPM, xmlmcErr := espXmlmc.Invoke("bpm", "processSpawn")
					if xmlmcErr != nil {
						//log.Fatal(xmlmcErr)
						logger(4, "Unable to invoke BPM for request ["+strNewCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
					}
					var xmlRespon xmlmcBPMSpawnedStruct

					errBPM := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
					if errBPM != nil {
						logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errBPM), false)
						return false, "No"
					}
					if xmlRespon.MethodResult != "ok" {
						logger(4, "Unable to invoke BPM: "+xmlRespon.State.ErrorRet, false)
					} else {
						//Now, associate spawned BPM to the new Request
						espXmlmc.SetParam("application", appServiceManager)
						espXmlmc.SetParam("entity", "Requests")
						espXmlmc.OpenElement("primaryEntityData")
						espXmlmc.OpenElement("record")
						espXmlmc.SetParam("h_pk_reference", strNewCallRef)
						espXmlmc.SetParam("h_bpm_id", xmlRespon.Identifier)
						espXmlmc.CloseElement("record")
						espXmlmc.CloseElement("primaryEntityData")

						XMLBPMUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityUpdateRecord")
						if xmlmcErr != nil {
							//log.Fatal(xmlmcErr)
							logger(4, "Unable to associated spawned BPM to request ["+strNewCallRef+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
						}
						var xmlRespon xmlmcResponse

						errBPMSpawn := xml.Unmarshal([]byte(XMLBPMUpdate), &xmlRespon)
						if errBPMSpawn != nil {
							logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errBPMSpawn), false)
							return false, "No"
						}
						if xmlRespon.MethodResult != "ok" {
							logger(4, "Unable to associate BPM to Request: "+xmlRespon.State.ErrorRet, false)
						}
					}
				}
			}

			// Now handle requests in an On Hold status
			if strStatus == "status.onHold" {
				espXmlmc.SetParam("requestId", strNewCallRef)
				espXmlmc.SetParam("onHoldUntil", strClosedDate)
				espXmlmc.SetParam("strReason", "Request imported from ServiceNow in an On Hold status. See Historical Request Updates for further information.")
				XMLBPM, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager+"/Requests", "holdRequest")
				if xmlmcErr != nil {
					//log.Fatal(xmlmcErr)
					logger(4, "Unable to place request on hold ["+strNewCallRef+"] : "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse

				errLogDate := xml.Unmarshal([]byte(XMLBPM), &xmlRespon)
				if errLogDate != nil {
					logger(4, "Unable to place request on hold ["+strNewCallRef+"] : "+fmt.Sprintf("%v", errLogDate), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(4, "Unable to place request on hold ["+strNewCallRef+"] : "+xmlRespon.State.ErrorRet, false)
				}
			}
		}
	} else {
		//-- DEBUG XML TO LOG FILE
		var XMLSTRING = espXmlmc.GetParam()
		logger(1, "Request Log XML "+fmt.Sprintf("%s", XMLSTRING), false)
		counters.Lock()
		counters.createdSkipped++
		counters.Unlock()
		espXmlmc.ClearParam()
		return true, "Dry Run"
	}

	//-- If request logged successfully :
	//Get the Call Diary Updates from ServiceNow and build the Historical Updates against the SM request
	if boolCallLoggedOK == true && strNewCallRef != "" {
		applyHistoricalUpdates(strNewCallRef, snCallID, fmt.Sprintf("%s", callMap["request_guid"]))
	}

	return boolCallLoggedOK, strNewCallRef
}

//convExtendedColName - takes old extended column name, returns new one (supply h_custom_a returns h_custom_1 for example)
//Split string in to array with _ as seperator
//Convert last array entry string character to Rune
//Convert Rune to Integer
//Subtract 96 from Integer
//Convert resulting Integer to String (numeric character), append to prefix and pass back
func convExtendedColName(oldColName string) string {
	arrColName := strings.Split(oldColName, "_")
	strNewColID := strconv.Itoa(int([]rune(arrColName[2])[0]) - 96)
	return "h_custom_" + strNewColID
}

//applyHistoricalUpdates - takes call diary records from ServiceNow, imports to Hornbill as Historical Updates
func applyHistoricalUpdates(newCallRef, snCallRef, snTaskSysID string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}

	//Connect to the JSON specified DB
	db, err := sqlx.Open(appDBDriver, connStrAppDB)
	defer db.Close()
	if err != nil {
		logger(4, " [DATABASE] Database Connection Error for Historical Updates: "+fmt.Sprintf("%v", err), false)
		return false
	}
	//Check connection is open
	err = db.Ping()
	if err != nil {
		logger(4, " [DATABASE] [PING] Database Connection Error for Historical Updates: "+fmt.Sprintf("%v", err), false)
		return false
	}
	//mutex.Lock()
	//build query
	sqlDiaryQuery := "SELECT element, value, sys_created_by, sys_created_on "
	sqlDiaryQuery = sqlDiaryQuery + " FROM sys_journal_field WHERE element_id = '" + snTaskSysID + "' ORDER BY sys_created_on ASC"
	if configDebug {
		logger(1, "[DATABASE] Running query for Historical Updates of call "+snCallRef+". Please wait...", false)
		logger(1, "[DATABASE] Diary Query: "+sqlDiaryQuery, false)
	}
	//mutex.Unlock()
	//Run Query
	rows, err := db.Queryx(sqlDiaryQuery)
	if err != nil {
		logger(4, " Database Query Error: "+fmt.Sprintf("%v", err), false)
		return false
	}
	rowCounter := 0
	//Process each call diary entry, insert in to Hornbill
	for rows.Next() {
		diaryEntry := make(map[string]interface{})
		err = rows.MapScan(diaryEntry)
		if err != nil {
			logger(4, "Unable to retrieve data from SQL query: "+fmt.Sprintf("%v", err), false)
		} else {
			rowCounter++
			//Update Time
			diaryTime := ""
			if diaryEntry["sys_created_on"] != nil {
				diaryTime = fmt.Sprintf("%+s", diaryEntry["sys_created_on"])
			}

			//Check for source/code/text having nil value
			diarySource := ""
			if diaryEntry["element"] != nil {
				diarySource = fmt.Sprintf("%+s", diaryEntry["element"]) + " (" + fmt.Sprintf("%+s", diaryEntry["sys_created_by"]) + ")"
			}

			diaryText := ""
			if diaryEntry["value"] != nil {
				diaryText = fmt.Sprintf("%+s", diaryEntry["value"])
				diaryText = html.EscapeString(diaryText)
			}

			diaryIndex := strconv.Itoa(rowCounter)

			espXmlmc.SetParam("application", appServiceManager)
			espXmlmc.SetParam("entity", "RequestHistoricUpdates")
			espXmlmc.OpenElement("primaryEntityData")
			espXmlmc.OpenElement("record")
			espXmlmc.SetParam("h_fk_reference", newCallRef)
			espXmlmc.SetParam("h_updatedate", diaryTime)
			espXmlmc.SetParam("h_updatebytype", "1")
			espXmlmc.SetParam("h_updateindex", diaryIndex)
			espXmlmc.SetParam("h_updateby", fmt.Sprintf("%+s", diaryEntry["sys_created_by"]))
			espXmlmc.SetParam("h_updatebyname", fmt.Sprintf("%+s", diaryEntry["sys_created_by"]))
			/*espXmlmc.SetParam("h_updatebygroup", fmt.Sprintf("%+s", diaryEntry["groupid"]))*/
			if diarySource != "" {
				espXmlmc.SetParam("h_actionsource", diarySource)
			}
			if diaryText != "" {
				espXmlmc.SetParam("h_description", diaryText)
			}
			espXmlmc.CloseElement("record")
			espXmlmc.CloseElement("primaryEntityData")

			//-- Check for Dry Run
			if configDryRun != true {
				XMLUpdate, xmlmcErr := espXmlmc.Invoke("data", "entityAddRecord")
				if xmlmcErr != nil {
					//log.Fatal(xmlmcErr)
					logger(3, "Unable to add Historical Call Diary Update: "+fmt.Sprintf("%v", xmlmcErr), false)
				}
				var xmlRespon xmlmcResponse
				errXMLMC := xml.Unmarshal([]byte(XMLUpdate), &xmlRespon)
				if errXMLMC != nil {
					logger(4, "Unable to read response from Hornbill instance:"+fmt.Sprintf("%v", errXMLMC), false)
				}
				if xmlRespon.MethodResult != "ok" {
					logger(3, "Unable to add Historical Call Diary Update: "+xmlRespon.State.ErrorRet, false)
				}
			} else {
				//-- DEBUG XML TO LOG FILE
				if configDebug {
					var XMLSTRING = espXmlmc.GetParam()
					logger(1, "Request Historical Update XML "+fmt.Sprintf("%s", XMLSTRING), false)
				}
				counters.Lock()
				counters.createdSkipped++
				counters.Unlock()
				espXmlmc.ClearParam()
				return true
			}
		}
	}
	defer rows.Close()
	return true
}

// getFieldValue --Retrieve field value from mapping via SQL record map
func getFieldValue(v string, u map[string]interface{}) string {
	fieldMap := v
	//-- Match $variable from String
	re1, err := regexp.Compile(`\[(.*?)\]`)
	if err != nil {
		color.Red("[ERROR] %v", err)
	}

	result := re1.FindAllString(fieldMap, 100)
	valFieldMap := ""
	//-- Loop Matches
	for _, val := range result {
		valFieldMap = ""
		valFieldMap = strings.Replace(val, "[", "", 1)
		valFieldMap = strings.Replace(valFieldMap, "]", "", 1)

		if valFieldMap == "callclass" {
			if fmt.Sprintf("%+s", u[valFieldMap]) == "sc_task" {
				fieldMap = strings.Replace(fieldMap, val, "Child Task of Parent Request!", 1)
			} else {
				fieldMap = strings.Replace(fieldMap, val, "", 1)
			}
		} else {
			if u[valFieldMap] != nil {

				if valField, ok := u[valFieldMap].(int64); ok {
					valFieldMap = strconv.FormatInt(valField, 10)
				} else {
					valFieldMap = fmt.Sprintf("%+s", u[valFieldMap])
				}

				if valFieldMap != "<nil>" {
					fieldMap = strings.Replace(fieldMap, val, valFieldMap, 1)
				}
			} else {
				fieldMap = strings.Replace(fieldMap, val, "", 1)
			}
		}
	}
	return fieldMap
}

//getSiteID takes the Call Record and returns a correct Site ID if one exists on the Instance
func getSiteID(callMap map[string]interface{}) (string, string) {
	siteID := ""
	siteNameMapping := fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_site_id"])
	siteName := getFieldValue(siteNameMapping, callMap)
	if siteName != "" {
		siteIsInCache, SiteIDCache := recordInCache(siteName, "Site")
		//-- Check if we have cached the site already
		if siteIsInCache {
			siteID = SiteIDCache
		} else {
			siteIsOnInstance, SiteIDInstance := searchSite(siteName)
			//-- If Returned set output
			if siteIsOnInstance {
				siteID = strconv.Itoa(SiteIDInstance)
			}
		}
	}
	return siteID, siteName
}

//getCallServiceID takes the Call Record and returns a correct Service ID if one exists on the Instance
func getCallServiceID(snService string) string {
	serviceID := ""
	serviceName := ""
	if mapGenericConf.ServiceMapping[snService] != nil {
		serviceName = fmt.Sprintf("%s", mapGenericConf.ServiceMapping[snService])

		if serviceName != "" {
			serviceID = getServiceID(serviceName)
		}
	}
	return serviceID
}

//getServiceID takes a Service Name string and returns a correct Service ID if one exists in the cache or on the Instance
func getServiceID(serviceName string) string {
	serviceID := ""
	if serviceName != "" {
		serviceIsInCache, ServiceIDCache := recordInCache(serviceName, "Service")
		//-- Check if we have cached the Service already
		if serviceIsInCache {
			serviceID = ServiceIDCache
		} else {
			serviceIsOnInstance, ServiceIDInstance := searchService(serviceName)
			//-- If Returned set output
			if serviceIsOnInstance {
				serviceID = strconv.Itoa(ServiceIDInstance)
			}
		}
	}
	return serviceID
}

//getCallPriorityID takes the Call Record and returns a correct Priority ID if one exists on the Instance
func getCallPriorityID(strPriorityName string) (string, string) {
	priorityID := ""
	if mapGenericConf.PriorityMapping[strPriorityName] != nil {
		strPriorityName = fmt.Sprintf("%s", mapGenericConf.PriorityMapping[strPriorityName])
		if strPriorityName != "" {
			priorityID = getPriorityID(strPriorityName)
		}
	}
	return priorityID, strPriorityName
}

//getPriorityID takes a Priority Name string and returns a correct Priority ID if one exists in the cache or on the Instance
func getPriorityID(priorityName string) string {
	priorityID := ""
	if priorityName != "" {
		priorityIsInCache, PriorityIDCache := recordInCache(priorityName, "Priority")
		//-- Check if we have cached the Priority already
		if priorityIsInCache {
			priorityID = PriorityIDCache
		} else {
			priorityIsOnInstance, PriorityIDInstance := searchPriority(priorityName)
			//-- If Returned set output
			if priorityIsOnInstance {
				priorityID = strconv.Itoa(PriorityIDInstance)
			}
		}
	}
	return priorityID
}

//getCallTeamID takes the Call Record and returns a correct Team ID if one exists on the Instance
func getCallTeamID(snTeamID string) (string, string) {
	teamID := ""
	teamName := ""
	if snImportConf.TeamMapping[snTeamID] != nil {
		teamName = fmt.Sprintf("%s", snImportConf.TeamMapping[snTeamID])
		if teamName != "" {
			teamID = getTeamID(teamName)
		}
	}
	return teamID, teamName
}

//getTeamID takes a Team Name string and returns a correct Team ID if one exists in the cache or on the Instance
func getTeamID(teamName string) string {
	teamID := ""
	if teamName != "" {
		teamIsInCache, TeamIDCache := recordInCache(teamName, "Team")
		//-- Check if we have cached the Team already
		if teamIsInCache {
			teamID = TeamIDCache
		} else {
			teamIsOnInstance, TeamIDInstance := searchTeam(teamName)
			//-- If Returned set output
			if teamIsOnInstance {
				teamID = TeamIDInstance
			}
		}
	}
	return teamID
}

//getCallCategoryID takes the Call Record and returns a correct Category ID if one exists on the Instance
func getCallCategoryID(callMap map[string]interface{}, categoryGroup string) (string, string) {
	categoryID := ""
	categoryString := ""
	categoryNameMapping := ""
	categoryCode := ""
	if categoryGroup == "Request" {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_category_id"])
		categoryCode = getFieldValue(categoryNameMapping, callMap)
		if snImportConf.CategoryMapping[categoryCode] != nil {
			//Get Category Code from JSON mapping
			categoryCode = fmt.Sprintf("%s", snImportConf.CategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - empty value
			categoryCode = ""
		}

	} else {
		categoryNameMapping = fmt.Sprintf("%v", mapGenericConf.CoreFieldMapping["h_closure_category_id"])
		categoryCode = getFieldValue(categoryNameMapping, callMap)
		if snImportConf.ResolutionCategoryMapping[categoryCode] != nil {
			//Get Category Code from JSON mapping
			categoryCode = fmt.Sprintf("%s", snImportConf.ResolutionCategoryMapping[categoryCode])
		} else {
			//Mapping doesn't exist - empty value
			categoryCode = ""
		}
	}
	if categoryCode != "" {
		categoryID, categoryString = getCategoryID(categoryCode, categoryGroup)
	}
	return categoryID, categoryString
}

//getCategoryID takes a Category Code string and returns a correct Category ID if one exists in the cache or on the Instance
func getCategoryID(categoryCode, categoryGroup string) (string, string) {
	categoryID := ""
	categoryString := ""
	if categoryCode != "" {
		categoryIsInCache, CategoryIDCache, CategoryNameCache := categoryInCache(categoryCode, categoryGroup+"Category")
		//-- Check if we have cached the Category already
		if categoryIsInCache {
			categoryID = CategoryIDCache
			categoryString = CategoryNameCache
		} else {
			categoryIsOnInstance, CategoryIDInstance, CategoryStringInstance := searchCategory(categoryCode, categoryGroup)
			//-- If Returned set output
			if categoryIsOnInstance {
				categoryID = CategoryIDInstance
				categoryString = CategoryStringInstance
			} else {
				logger(4, "[CATEGORY] "+categoryGroup+" Category ["+categoryCode+"] is not on instance.", false)
			}
		}
	}
	return categoryID, categoryString
}

//doesAnalystExist takes an Analyst ID string and returns a true if one exists in the cache or on the Instance
func doesAnalystExist(analystID string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	boolAnalystExists := false
	if analystID != "" {
		analystIsInCache, strReturn := recordInCache(analystID, "Analyst")
		//-- Check if we have cached the Analyst already
		if analystIsInCache && strReturn != "" {
			boolAnalystExists = true
		} else {
			//Get Analyst Info
			espXmlmc.SetParam("userId", analystID)

			XMLAnalystSearch, xmlmcErr := espXmlmc.Invoke("admin", "userGetInfo")
			if xmlmcErr != nil {
				logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
			}

			var xmlRespon xmlmcAnalystListResponse
			err := xml.Unmarshal([]byte(XMLAnalystSearch), &xmlRespon)
			if err != nil {
				logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+fmt.Sprintf("%v", err), false)
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Analyst most likely does not exist
					logger(4, "Unable to Search for Request Owner ["+analystID+"]: "+xmlRespon.State.ErrorRet, false)
				} else {
					//-- Check Response
					if xmlRespon.AnalystFullName != "" {
						boolAnalystExists = true
						//-- Add Analyst to Cache
						var newAnalystForCache analystListStruct
						newAnalystForCache.AnalystID = analystID
						newAnalystForCache.AnalystName = xmlRespon.AnalystFullName
						analystNamedMap := []analystListStruct{newAnalystForCache}
						mutexAnalysts.Lock()
						analysts = append(analysts, analystNamedMap...)
						mutexAnalysts.Unlock()
					}
				}
			}
		}
	}
	return boolAnalystExists
}

//doesCustomerExist takes a Customer ID string and returns a true if one exists in the cache or on the Instance
func doesCustomerExist(customerID string) bool {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false
	}
	boolCustomerExists := false
	if customerID != "" {
		customerIsInCache, strReturn := recordInCache(customerID, "Customer")
		//-- Check if we have cached the Analyst already
		if customerIsInCache && strReturn != "" {
			boolCustomerExists = true
		} else {
			//Get Analyst Info
			espXmlmc.SetParam("customerId", customerID)
			espXmlmc.SetParam("customerType", snImportConf.CustomerType)
			XMLCustomerSearch, xmlmcErr := espXmlmc.Invoke("apps/"+appServiceManager, "shrGetCustomerDetails")
			if xmlmcErr != nil {
				logger(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", xmlmcErr), true)
			}

			var xmlRespon xmlmcCustomerListResponse
			err := xml.Unmarshal([]byte(XMLCustomerSearch), &xmlRespon)
			if err != nil {
				logger(4, "Unable to Search for Customer ["+customerID+"]: "+fmt.Sprintf("%v", err), false)
			} else {
				if xmlRespon.MethodResult != "ok" {
					//Customer most likely does not exist
					logger(4, "Unable to Search for Customer ["+customerID+"]: "+xmlRespon.State.ErrorRet, false)
				} else {
					//-- Check Response
					if xmlRespon.CustomerFirstName != "" {
						boolCustomerExists = true
						//-- Add Customer to Cache
						var newCustomerForCache customerListStruct
						newCustomerForCache.CustomerID = customerID
						newCustomerForCache.CustomerName = xmlRespon.CustomerFirstName + " " + xmlRespon.CustomerLastName
						customerNamedMap := []customerListStruct{newCustomerForCache}
						mutexCustomers.Lock()
						customers = append(customers, customerNamedMap...)
						mutexCustomers.Unlock()
					}
				}
			}
		}
	}
	return boolCustomerExists
}

// recordInCache -- Function to check if passed-thorugh record name has been cached
// if so, pass back the Record ID
func recordInCache(recordName, recordType string) (bool, string) {
	boolReturn := false
	strReturn := ""
	switch recordType {
	case "Service":
		//-- Check if record in Service Cache
		mutexServices.Lock()
		for _, service := range services {
			if service.ServiceName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(service.ServiceID)
			}
		}
		mutexServices.Unlock()
	case "Priority":
		//-- Check if record in Priority Cache
		mutexPriorities.Lock()
		for _, priority := range priorities {
			if priority.PriorityName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(priority.PriorityID)
			}
		}
		mutexPriorities.Unlock()
	case "Site":
		//-- Check if record in Site Cache
		mutexSites.Lock()
		for _, site := range sites {
			if site.SiteName == recordName {
				boolReturn = true
				strReturn = strconv.Itoa(site.SiteID)
			}
		}
		mutexSites.Unlock()
	case "Team":
		//-- Check if record in Team Cache
		mutexTeams.Lock()
		for _, team := range teams {
			if team.TeamName == recordName {
				boolReturn = true
				strReturn = team.TeamID
			}
		}
		mutexTeams.Unlock()
	case "Analyst":
		//-- Check if record in Analyst Cache
		mutexAnalysts.Lock()
		for _, analyst := range analysts {
			if analyst.AnalystID == recordName {
				boolReturn = true
				strReturn = analyst.AnalystName
			}
		}
		mutexAnalysts.Unlock()
	case "Customer":
		//-- Check if record in Customer Cache
		mutexCustomers.Lock()
		for _, customer := range customers {
			if customer.CustomerID == recordName {
				boolReturn = true
				strReturn = customer.CustomerName
			}
		}
		mutexCustomers.Unlock()
	}
	return boolReturn, strReturn
}

// categoryInCache -- Function to check if passed-thorugh category been cached
// if so, pass back the Category ID and Full Name
func categoryInCache(recordName, recordType string) (bool, string, string) {
	boolReturn := false
	idReturn := ""
	strReturn := ""
	switch recordType {
	case "RequestCategory":
		//-- Check if record in Category Cache
		mutexCategories.Lock()
		for _, category := range categories {
			if category.CategoryCode == recordName {
				boolReturn = true
				idReturn = category.CategoryID
				strReturn = category.CategoryName
			}
		}
		mutexCategories.Unlock()
	case "ClosureCategory":
		//-- Check if record in Category Cache
		mutexCloseCategories.Lock()
		for _, category := range closeCategories {
			if category.CategoryCode == recordName {
				boolReturn = true
				idReturn = category.CategoryID
				strReturn = category.CategoryName
			}
		}
		mutexCloseCategories.Unlock()
	}
	return boolReturn, idReturn, strReturn
}

// seachSite -- Function to check if passed-through  site  name is on the instance
func searchSite(siteName string) (bool, int) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}

	boolReturn := false
	intReturn := 0
	//-- ESP Query for site
	espXmlmc.SetParam("entity", "Site")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_site_name", siteName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLSiteSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Site: "+fmt.Sprintf("%v", xmlmcErr), false)
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcSiteListResponse

	err = xml.Unmarshal([]byte(XMLSiteSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Site: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Site: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.SiteName != "" {
				if strings.ToLower(xmlRespon.SiteName) == strings.ToLower(siteName) {
					intReturn = xmlRespon.SiteID
					boolReturn = true
					//-- Add Site to Cache
					var newSiteForCache siteListStruct
					newSiteForCache.SiteID = intReturn
					newSiteForCache.SiteName = siteName
					siteNamedMap := []siteListStruct{newSiteForCache}
					mutexSites.Lock()
					sites = append(sites, siteNamedMap...)
					mutexSites.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}

// seachPriority -- Function to check if passed-through priority name is on the instance
func searchPriority(priorityName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	//-- ESP Query for Priority
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}

	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Priority")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_priorityname", priorityName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLPrioritySearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Priority: "+fmt.Sprintf("%v", xmlmcErr), false)
		return boolReturn, intReturn
		//log.Fatal(xmlmcErr)
	}
	var xmlRespon xmlmcPriorityListResponse

	err = xml.Unmarshal([]byte(XMLPrioritySearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Priority: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Priority: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.PriorityName != "" {
				if strings.ToLower(xmlRespon.PriorityName) == strings.ToLower(priorityName) {
					intReturn = xmlRespon.PriorityID
					boolReturn = true
					//-- Add Priority to Cache
					var newPriorityForCache priorityListStruct
					newPriorityForCache.PriorityID = intReturn
					newPriorityForCache.PriorityName = priorityName
					priorityNamedMap := []priorityListStruct{newPriorityForCache}
					mutexPriorities.Lock()
					priorities = append(priorities, priorityNamedMap...)
					mutexPriorities.Unlock()
				}
			}
		}
	}
	return boolReturn, intReturn
}

// seachService -- Function to check if passed-through service name is on the instance
func searchService(serviceName string) (bool, int) {
	boolReturn := false
	intReturn := 0
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, 0
	}

	//-- ESP Query for service
	espXmlmc.SetParam("application", appServiceManager)
	espXmlmc.SetParam("entity", "Services")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_servicename", serviceName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLServiceSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Service: "+fmt.Sprintf("%v", xmlmcErr), false)
		//log.Fatal(xmlmcErr)
		return boolReturn, intReturn
	}
	var xmlRespon xmlmcServiceListResponse

	err = xml.Unmarshal([]byte(XMLServiceSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Service: "+fmt.Sprintf("%v", err), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Service: "+xmlRespon.State.ErrorRet, false)
		} else {
			//-- Check Response
			if xmlRespon.ServiceName != "" {
				if strings.ToLower(xmlRespon.ServiceName) == strings.ToLower(serviceName) {
					intReturn = xmlRespon.ServiceID
					boolReturn = true
					//-- Add Service to Cache
					var newServiceForCache serviceListStruct
					newServiceForCache.ServiceID = intReturn
					newServiceForCache.ServiceName = serviceName
					newServiceForCache.ServiceBPMIncident = xmlRespon.BPMIncident
					newServiceForCache.ServiceBPMService = xmlRespon.BPMService
					newServiceForCache.ServiceBPMChange = xmlRespon.BPMChange
					newServiceForCache.ServiceBPMProblem = xmlRespon.BPMProblem
					newServiceForCache.ServiceBPMKnownError = xmlRespon.BPMKnownError
					serviceNamedMap := []serviceListStruct{newServiceForCache}
					mutexServices.Lock()
					services = append(services, serviceNamedMap...)
					mutexServices.Unlock()
				}
			}
		}
	}
	//Return Service ID once cached - we can now use this in the calling function to get all details from cache
	return boolReturn, intReturn
}

// searchTeam -- Function to check if passed-through support team name is on the instance
func searchTeam(teamName string) (bool, string) {
	boolReturn := false
	strReturn := ""
	//-- ESP Query for team
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create connection"
	}

	espXmlmc.SetParam("entity", "Groups")
	espXmlmc.SetParam("matchScope", "all")
	espXmlmc.OpenElement("searchFilter")
	espXmlmc.SetParam("h_name", teamName)
	espXmlmc.CloseElement("searchFilter")
	espXmlmc.SetParam("maxResults", "1")

	XMLTeamSearch, xmlmcErr := espXmlmc.Invoke("data", "entityBrowseRecords")
	if xmlmcErr != nil {
		logger(4, "Unable to Search for Team: "+fmt.Sprintf("%v", xmlmcErr), true)
		//log.Fatal(xmlmcErr)
		return boolReturn, strReturn
	}
	var xmlRespon xmlmcTeamListResponse

	err = xml.Unmarshal([]byte(XMLTeamSearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Search for Team: "+fmt.Sprintf("%v", err), true)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for Team: "+xmlRespon.State.ErrorRet, true)
		} else {
			//-- Check Response
			if xmlRespon.TeamName != "" {
				if strings.ToLower(xmlRespon.TeamName) == strings.ToLower(teamName) {
					strReturn = xmlRespon.TeamID
					boolReturn = true
					//-- Add Team to Cache
					var newTeamForCache teamListStruct
					newTeamForCache.TeamID = strReturn
					newTeamForCache.TeamName = teamName
					teamNamedMap := []teamListStruct{newTeamForCache}
					mutexTeams.Lock()
					teams = append(teams, teamNamedMap...)
					mutexTeams.Unlock()
				}
			}
		}
	}
	return boolReturn, strReturn
}

// seachCategory -- Function to check if passed-through support category name is on the instance
func searchCategory(categoryCode, categoryGroup string) (bool, string, string) {
	espXmlmc, err := NewEspXmlmcSession()
	if err != nil {
		return false, "Unable to create connection", ""
	}

	boolReturn := false
	idReturn := ""
	strReturn := ""
	//-- ESP Query for category
	espXmlmc.SetParam("codeGroup", categoryGroup)
	espXmlmc.SetParam("code", categoryCode)
	var XMLSTRING = espXmlmc.GetParam()
	XMLCategorySearch, xmlmcErr := espXmlmc.Invoke("data", "profileCodeLookup")
	if xmlmcErr != nil {
		logger(4, "XMLMC API Invoke Failed for "+categoryGroup+" Category ["+categoryCode+"]: "+fmt.Sprintf("%v", xmlmcErr), false)
		logger(1, "Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
		return boolReturn, idReturn, strReturn
	}
	var xmlRespon xmlmcCategoryListResponse

	err = xml.Unmarshal([]byte(XMLCategorySearch), &xmlRespon)
	if err != nil {
		logger(4, "Unable to unmarshal response for "+categoryGroup+" Category: "+fmt.Sprintf("%v", err), false)
		logger(1, "Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
	} else {
		if xmlRespon.MethodResult != "ok" {
			logger(4, "Unable to Search for "+categoryGroup+" Category ["+categoryCode+"]: ["+fmt.Sprintf("%v", xmlRespon.MethodResult)+"] "+xmlRespon.State.ErrorRet, false)
			logger(1, "Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
		} else {
			//-- Check Response
			if xmlRespon.CategoryName != "" {
				strReturn = xmlRespon.CategoryName
				idReturn = xmlRespon.CategoryID
				if configDebug {
					logger(1, "[CATEGORY] [SUCCESS] Methodcall result OK for "+categoryGroup+" Category ["+categoryCode+"] : ["+strReturn+"]", false)
				}
				boolReturn = true
				//-- Add Category to Cache
				var newCategoryForCache categoryListStruct
				newCategoryForCache.CategoryID = idReturn
				newCategoryForCache.CategoryCode = categoryCode
				newCategoryForCache.CategoryName = strReturn
				categoryNamedMap := []categoryListStruct{newCategoryForCache}
				switch categoryGroup {
				case "Request":
					mutexCategories.Lock()
					categories = append(categories, categoryNamedMap...)
					mutexCategories.Unlock()
				case "Closure":
					mutexCloseCategories.Lock()
					closeCategories = append(closeCategories, categoryNamedMap...)
					mutexCloseCategories.Unlock()
				}
			} else {
				logger(3, "[CATEGORY] [FAIL] Methodcall result OK for "+categoryGroup+" Category ["+categoryCode+"] but category name blank: ["+xmlRespon.CategoryID+"] ["+xmlRespon.CategoryName+"]", false)
				if configDebug {
					logger(1, "[CATEGORY] [FAIL] Category Search XML "+fmt.Sprintf("%s", XMLSTRING), false)
				}
			}
		}
	}
	return boolReturn, idReturn, strReturn
}

//loadConfig -- Function to Load Configruation File
func loadConfig() (snImportConfStruct, bool) {
	boolLoadConf := true
	//-- Check Config File File Exists
	cwd, _ := os.Getwd()
	configurationFilePath := cwd + "/" + configFileName
	logger(1, "Loading Config File: "+configurationFilePath, false)
	if _, fileCheckErr := os.Stat(configurationFilePath); os.IsNotExist(fileCheckErr) {
		logger(4, "No Configuration File", true)
		os.Exit(102)
	}
	//-- Load Config File
	file, fileError := os.Open(configurationFilePath)
	//-- Check For Error Reading File
	if fileError != nil {
		logger(4, "Error Opening Configuration File: "+fmt.Sprintf("%v", fileError), true)
		boolLoadConf = false
	}

	//-- New Decoder
	decoder := json.NewDecoder(file)
	//-- New Var based on snImportConfStruct
	edbConf := snImportConfStruct{}
	//-- Decode JSON
	err := decoder.Decode(&edbConf)
	//-- Error Checking
	if err != nil {
		logger(4, "Error Decoding Configuration File: "+fmt.Sprintf("%v", err), true)
		boolLoadConf = false
	}
	//-- Return New Config
	return edbConf, boolLoadConf
}

//login -- XMLMC Login
//-- start ESP user session
func login() bool {
	logger(1, "Logging Into: "+snImportConf.HBConf.URL, false)
	logger(1, "UserName: "+snImportConf.HBConf.UserName, false)
	espXmlmc = apiLib.NewXmlmcInstance(snImportConf.HBConf.URL)

	espXmlmc.SetParam("userId", snImportConf.HBConf.UserName)
	espXmlmc.SetParam("password", base64.StdEncoding.EncodeToString([]byte(snImportConf.HBConf.Password)))
	XMLLogin, xmlmcErr := espXmlmc.Invoke("session", "userLogon")
	if xmlmcErr != nil {
		log.Fatal(xmlmcErr)
	}

	var xmlRespon xmlmcResponse
	err := xml.Unmarshal([]byte(XMLLogin), &xmlRespon)
	if err != nil {
		logger(4, "Unable to Login: "+fmt.Sprintf("%v", err), true)
		return false
	}
	if xmlRespon.MethodResult != "ok" {
		logger(4, "Unable to Login: "+xmlRespon.State.ErrorRet, true)
		return false
	}
	espLogger("---- ServiceNow Task Import Utility V"+fmt.Sprintf("%v", version)+" ----", "debug")
	espLogger("Logged In As: "+snImportConf.HBConf.UserName, "debug")
	return true
}

//logout -- XMLMC Logout
//-- Adds details to log file, ends user ESP session
func logout() {
	//-- End output
	espLogger("Requests Logged: "+fmt.Sprintf("%d", counters.created), "debug")
	espLogger("Requests Skipped: "+fmt.Sprintf("%d", counters.createdSkipped), "debug")
	espLogger("Time Taken: "+fmt.Sprintf("%v", endTime), "debug")
	espLogger("---- ServiceNow Task Import Complete ---- ", "debug")
	logger(1, "Logout", true)
	espXmlmc.Invoke("session", "userLogoff")
}

//buildConnectionString -- Build the connection string for the SQL driver
func buildConnectionString() string {
	connectString := ""

	//Build
	if appDBDriver == "" || snImportConf.SNAppDBConf.Server == "" || snImportConf.SNAppDBConf.Database == "" || snImportConf.SNAppDBConf.UserName == "" || snImportConf.SNAppDBConf.Port == 0 {
		logger(4, "ServiceNow Database configuration not set.", true)
		return ""
	}
	switch appDBDriver {
	case "mssql":
		connectString = "server=" + snImportConf.SNAppDBConf.Server
		connectString = connectString + ";database=" + snImportConf.SNAppDBConf.Database
		connectString = connectString + ";user id=" + snImportConf.SNAppDBConf.UserName
		connectString = connectString + ";password=" + snImportConf.SNAppDBConf.Password
		if snImportConf.SNAppDBConf.Encrypt == false {
			connectString = connectString + ";encrypt=disable"
		}
		if snImportConf.SNAppDBConf.Port != 0 {
			var dbPortSetting string
			dbPortSetting = strconv.Itoa(snImportConf.SNAppDBConf.Port)
			connectString = connectString + ";port=" + dbPortSetting
		}
	case "mysql":
		connectString = snImportConf.SNAppDBConf.UserName + ":" + snImportConf.SNAppDBConf.Password
		connectString = connectString + "@tcp(" + snImportConf.SNAppDBConf.Server + ":"
		if snImportConf.SNAppDBConf.Port != 0 {
			var dbPortSetting string
			dbPortSetting = strconv.Itoa(snImportConf.SNAppDBConf.Port)
			connectString = connectString + dbPortSetting
		} else {
			connectString = connectString + "3306"
		}
		connectString = connectString + ")/" + snImportConf.SNAppDBConf.Database

	case "mysql320":
		var dbPortSetting string
		dbPortSetting = strconv.Itoa(snImportConf.SNAppDBConf.Port)
		connectString = "tcp:" + snImportConf.SNAppDBConf.Server + ":" + dbPortSetting
		connectString = connectString + "*" + snImportConf.SNAppDBConf.Database + "/" + snImportConf.SNAppDBConf.UserName + "/" + snImportConf.SNAppDBConf.Password
	}
	return connectString
}

// logger -- function to append to the current log file
func logger(t int, s string, outputtoCLI bool) {
	cwd, _ := os.Getwd()
	logPath := cwd + "/log"
	logFileName := logPath + "/SN_Task_Import_" + timeNow + ".log"

	//-- If Folder Does Not Exist then create it
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		err := os.Mkdir(logPath, 0777)
		if err != nil {
			color.Red("Error Creating Log Folder %q: %s \r", logPath, err)
			os.Exit(101)
		}
	}

	//-- Open Log File
	f, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	// don't forget to close it
	defer f.Close()
	if err != nil {
		color.Red("Error Creating Log File %q: %s \n", logFileName, err)
		os.Exit(100)
	}
	var errorLogPrefix string
	//-- Create Log Entry
	switch t {
	case 1:
		errorLogPrefix = "[DEBUG] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 2:
		errorLogPrefix = "[MESSAGE] "
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 3:
		if outputtoCLI {
			color.Set(color.FgGreen)
			defer color.Unset()
		}
	case 4:
		errorLogPrefix = "[ERROR] "
		if outputtoCLI {
			color.Set(color.FgRed)
			defer color.Unset()
		}
	case 5:
		errorLogPrefix = "[WARNING]"
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	case 6:
		if outputtoCLI {
			color.Set(color.FgYellow)
			defer color.Unset()
		}
	}

	if outputtoCLI {
		fmt.Printf("%v \n", errorLogPrefix+s)
	}
	mutexLogging.Lock()
	// assign the file to the standard logger
	log.SetOutput(f)
	//Write the log entry
	log.Println(errorLogPrefix + s)
	mutexLogging.Unlock()
}

// espLogger -- Log to ESP
func espLogger(message string, severity string) {

	espXmlmc.SetParam("fileName", "SN_Task_Import")
	espXmlmc.SetParam("group", "general")
	espXmlmc.SetParam("severity", severity)
	espXmlmc.SetParam("message", message)
	espXmlmc.Invoke("system", "logMessage")
}

// SetInstance sets the Zone and Instance config from the passed-through strZone and instanceID values
func SetInstance(strZone string, instanceID string) {
	//-- Set Zone
	SetZone(strZone)
	//-- Set Instance
	xmlmcInstanceConfig.instance = instanceID
	return
}

// SetZone - sets the Instance Zone to Overide current live zone
func SetZone(zone string) {
	xmlmcInstanceConfig.zone = zone
	return
}

// getInstanceURL -- Function to build XMLMC End Point
func getInstanceURL() string {
	xmlmcInstanceConfig.url = "https://"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.zone
	xmlmcInstanceConfig.url += "api.hornbill.com/"
	xmlmcInstanceConfig.url += xmlmcInstanceConfig.instance
	xmlmcInstanceConfig.url += "/xmlmc/"
	return xmlmcInstanceConfig.url
}

//NewEspXmlmcSession - New Xmlmc Session variable (Cloned Session)
func NewEspXmlmcSession() (*apiLib.XmlmcInstStruct, error) {
	time.Sleep(150 * time.Millisecond)
	espXmlmcLocal := apiLib.NewXmlmcInstance(snImportConf.HBConf.URL)
	espXmlmcLocal.SetSessionID(espXmlmc.GetSessionID())
	return espXmlmcLocal, nil
}