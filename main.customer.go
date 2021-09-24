package main

import (
	"encoding/xml"
	"fmt"
	apiLib "github.com/hornbill/goApiLib"
	"strconv"
	"strings"

	"github.com/hornbill/pb"
)

type xmlmcUserListResponse struct {
	Params struct {
		RowData struct {
			Row []userAccountStruct `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateJSONStruct `xml:"state"`
}

type userAccountStruct struct {
	HUserID     string `xml:"h_user_id"`
	HLoginID    string `xml:"h_login_id"`
	HEmployeeID string `xml:"h_employee_id"`
	HName       string `xml:"h_name"`
	HFirstName  string `xml:"h_first_name"`
	HLastName   string `xml:"h_last_name"`
	HEmail      string `xml:"h_email"`
	HAttrib1    string `xml:"h_attrib_1"`
	HClass      string `xml:"h_class"`
}
type xmlmcCountResponse struct {
	Params struct {
		RowData struct {
			Row []struct {
				Count string `xml:"count"`
			} `xml:"row"`
		} `xml:"rowData"`
	} `xml:"params"`
	State stateJSONStruct `xml:"state"`
}
type stateJSONStruct struct {
	Code      string `xml:"code"`
	Service   string `xml:"service"`
	Operation string `xml:"operation"`
	Error     string `xml:"error"`
}

func loadUsers() {
	//-- Init One connection to Hornbill to load all data
	logger(1, "Loading Users from Hornbill", false)

	count := getCount("getUserAccountsList")
	logger(1, "getUserAccountsList Count: "+strconv.FormatUint(count, 10), false)
	getUserAccountList(count)

	logger(1, "Users Loaded: "+strconv.Itoa(len(customers)), false)
	logger(1, "Analysts Loaded: "+strconv.Itoa(len(analysts)), false)
}

func getUserAccountList(count uint64) {
	var loopCount uint64
	//-- Init Map
	//-- Load Results in pages of pageSize
	bar := pb.StartNew(int(count))
	for loopCount < count {
		logger(1, "Loading User Accounts List Offset: "+fmt.Sprintf("%d", loopCount)+"\n", false)

		espXmlmc.SetParam("application", "com.hornbill.core")
		espXmlmc.SetParam("queryName", "getUserAccountsList")
		espXmlmc.OpenElement("queryParams")
		espXmlmc.SetParam("rowstart", strconv.FormatUint(loopCount, 10))
		espXmlmc.SetParam("limit", strconv.Itoa(pageSize))
		espXmlmc.CloseElement("queryParams")
		RespBody, xmlmcErr := espXmlmc.Invoke("data", "queryExec")

		var JSONResp xmlmcUserListResponse
		if xmlmcErr != nil {
			logger(4, "Unable to Query Accounts List "+xmlmcErr.Error(), false)
			break
		}
		err := xml.Unmarshal([]byte(RespBody), &JSONResp)
		if err != nil {
			logger(4, "Unable to Query Accounts List "+err.Error(), false)
			break
		}
		if JSONResp.State.Error != "" {
			logger(4, "Unable to Query Accounts List "+JSONResp.State.Error, false)
			break
		}
		//-- Push into Map
		for index := range JSONResp.Params.RowData.Row {
			var newCustomerForCache customerListStruct
			switch snImportConf.CustomerUniqueColumn {
			case "h_user_id":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HUserID
				}
			case "h_employee_id":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HEmployeeID
				}
			case "h_login_id":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HLoginID
				}
			case "h_email":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HEmail
				}
			case "h_name":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HName
				}
			case "h_attrib_1":
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HAttrib1
				}
			default:
				{
					newCustomerForCache.CustomerID = JSONResp.Params.RowData.Row[index].HUserID
				}
			}
			newCustomerForCache.CustomerHandle = JSONResp.Params.RowData.Row[index].HUserID
			newCustomerForCache.CustomerName = JSONResp.Params.RowData.Row[index].HFirstName + " " + JSONResp.Params.RowData.Row[index].HLastName
			//fmt.Println(newCustomerForCache.CustomerID + "::" + newCustomerForCache.CustomerHandle + "--" + newCustomerForCache.CustomerName)
			customerNamedMap := []customerListStruct{newCustomerForCache}
			mutexCustomers.Lock()
			customers = append(customers, customerNamedMap...)
			mutexCustomers.Unlock()

			if JSONResp.Params.RowData.Row[index].HClass == "1" {
				var newAnalystForCache analystListStruct
				switch snImportConf.AnalystUniqueColumn {
				case "h_user_id":
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HUserID
					}
				case "h_employee_id":
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HEmployeeID
					}
				case "h_login_id":
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HLoginID
					}
				case "h_email":
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HEmail
					}
				case "h_name":
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HName
					}
				case "h_attrib_1":
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HAttrib1
					}
				default:
					{
						newAnalystForCache.AnalystID = JSONResp.Params.RowData.Row[index].HUserID
					}
				}
				newAnalystForCache.AnalystHandle = JSONResp.Params.RowData.Row[index].HUserID
				newAnalystForCache.AnalystName = JSONResp.Params.RowData.Row[index].HName
				analystNamedMap := []analystListStruct{newAnalystForCache}
				mutexAnalysts.Lock()
				analysts = append(analysts, analystNamedMap...)
				mutexAnalysts.Unlock()

			}

		}

		// Add 100
		loopCount += uint64(pageSize)
		bar.Add(len(JSONResp.Params.RowData.Row))
		//-- Check for empty result set
		if len(JSONResp.Params.RowData.Row) == 0 {
			break
		}
	}
	bar.FinishPrint("Accounts Loaded  \n")
}

func getCount(query string) uint64 {
	espXmlmc := apiLib.NewXmlmcInstance(snImportConf.HBConf.InstanceID)
	espXmlmc.SetAPIKey(snImportConf.HBConf.APIKey)

	espXmlmc.SetParam("application", "com.hornbill.core")
	espXmlmc.SetParam("queryName", query)
	espXmlmc.OpenElement("queryParams")
	espXmlmc.SetParam("getCount", "true")
	espXmlmc.CloseElement("queryParams")

	RespBody, xmlmcErr := espXmlmc.Invoke("data", "queryExec")

	var JSONResp xmlmcCountResponse
	if xmlmcErr != nil {
		logger(4, "Unable to run Query ["+query+"] "+xmlmcErr.Error(), false)
		return 0
	}
	err := xml.Unmarshal([]byte(RespBody), &JSONResp)
	if err != nil {
		logger(4, "Unable to run Query ["+query+"] "+err.Error(), false)
		return 0
	}
	if JSONResp.State.Error != "" {
		logger(4, "Unable to run Query ["+query+"] "+JSONResp.State.Error, false)
		return 0
	}

	//-- return Count
	count, errC := strconv.ParseUint(JSONResp.Params.RowData.Row[0].Count, 10, 16)
	//-- Check for Error
	if errC != nil {
		logger(4, "Unable to get Count for Query ["+query+"] "+err.Error(), false)
		return 0
	}
	return count
}

func getUserID(userID string) (UserID, userURN, userName string) {
	if userID != "" && userID != "<nil>" && userID != "__clear__" {
		mutexCustomers.Lock()
		for _, customer := range customers {
			if strings.EqualFold(customer.CustomerID, userID) {
				UserID = customer.CustomerHandle
				userName = customer.CustomerName
				break
			}
		}
		mutexCustomers.Unlock()
	}
	if userName != "" {
		userURN = "urn:sys:0:" + userName + ":" + UserID
	} else {
		UserID = userID
	}
	logger(1, "User Mapping:"+UserID+":"+userID+":"+userName+":"+userURN, false)
	return
}
