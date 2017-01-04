### ServiceNow Task Import [GO](https://golang.org/) - Import Script to Hornbill

### Quick links
- [Overview](#overview)
- [Installation](#Installation)
- [Configuration](Cconfiguration)
    - [HBConfig](#HBConfig)
    - [ServiceNow Database Configuration](#SNAppDBConf)
    - [Task Class Specific Configuration](#ConfCallClass)
    - [Activity Task Specific Configuration](#ConfActivities)
    - [Team/Support Group Mapping](#TeamMapping)
    - [Category Mapping](#CategoryMapping)
    - [Resolution Category Mapping](#ResolutionCategoryMapping)
- [Execute](#execute)
- [Testing](testing)
- [Logging](#logging)
- [Error Codes](#error codes)

# Overview
This tool provides functionality to allow the import of Task/Request data from ServiceNow in to Hornbill Service Manager.

The following tasks are carried out when the tool is executed:
* ServiceNow task data is extracted as per your specification, as outlined in the Configuration section of this document;
* New requests are raised on Service Manager using the extracted task data and associated mapping specifications;
* ServiceNow Task diary entries are imported as Historic Updates against the new Service Manager Requests;
* ServiceNow Task attachments are attached to the relevant imported Service Manager request;
* Previous parent-child relationships between ServiceNow Tasks are re-created by linking the newly-imported Service Manager requests with their previous parents/children;
* ServiceNow Approval Tasks are imported as Hornbill Activity Tasks, and associated with the relevant imported Requests.

#### IMPORTANT!
Importing ServiceNow task data and associated file attachments will consume your subscribed Hornbill storage. Please check your Administration console and your ServiceNow data to ensure that you have enough subscribed storage available before running this import.

# Installation

#### Windows
* Download the archive containing the import executables
* Extract zip into a folder you would like the application to run from e.g. `C:\servicenow_request_import\`
* Open '''conf.json''' and add in the necessary configuration
* Open Command Line Prompt as Administrator
* Change Directory to the folder containing the extracted files `C:\servicenow_request_import\`
* Run the command relevant to the computer you are running this on:
* - For 32 Bit Windows Machines : servicenow_request_import_w32.exe -dryrun=true
* - For 64 Bit Windows Machines : servicenow_request_import_w64.exe -dryrun=true

# Configuration

Example JSON File:

```json
{
  "HBConf": {
    "UserName": "Hornbill Instance Username",
    "Password": "Hornbill Instance Password",
    "InstanceID": "Hornbill Instance ID (case sensitive)"
  },
  "SNAppDBConf": {
    "Driver": "mysql",
    "Server": "IP Address of Database Server",
    "Database": "servicenow_dbname",
    "UserName": "Database User ID",
    "Password": "Database Password",
    "Port": 3306,
    "Encrypt": false
  },
  "CustomerType": "0",
  "ConfIncident": {
    "Import":false,
    "CallClass": "Incident",
    "DefaultTeam":"Service Desk",
    "DefaultPriority":"Low",
    "DefaultService":"ServiceNow Historic Requests",
    "SQLStatement":{
      "0":"SELECT task.sys_id AS request_guid, task.sys_class_name AS callclass, task.number AS callref, ",
      "1":"task.made_sla, task.opened_at AS logdate, task.u_desk_visit, u_symptoms.u_name AS symptom_name, ",
      "2":"task.short_description, task.description, task.u_category, task.category_1, task.contact_type, ",
      "3":"task.u_resolved_at, task.closed_at, task.close_code, task.close_notes, task.sys_created_by AS createdby_username, ",
      "4":"core_company.name AS company_name, cmdb_ci.name AS service_name, dept.name AS department, ",
      "5":"task.u_first_line_fix, parent_task.number AS parent_task_ref, ",
      "6":"(SELECT label FROM sys_choice where name = 'incident' AND element = 'incident_state' AND value = task.incident_state) AS incident_state, ",
      "7":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state, ",
      "8":"(SELECT name from cmdb_ci where sys_id = task.u_close_ci) AS close_ci, ",
      "9":"(SELECT user_name FROM sys_user where sys_id = task.opened_by) AS loggedby, ",
      "10":"(SELECT user_name FROM sys_user where sys_id = task.assigned_to) AS owner_username, ",
      "11":"(SELECT name FROM sys_user where sys_id = task.assigned_to) AS owner_name, ",
      "12":"(SELECT name FROM sys_user_group where sys_id = task.assignment_group) AS support_group, ",
      "13":"(SELECT user_name FROM sys_user WHERE sys_id = task.a_ref_1) AS incident_customer_username, ",
      "14":"(SELECT name FROM sys_user where sys_id = task.a_ref_1) AS incident_customer_name, ",
      "15":"(SELECT label FROM sys_choice where name = 'incident' AND element = 'priority' AND value = task.priority) AS priority, ",
      "16":"(SELECT label FROM sys_choice where name = 'incident' AND element = 'impact' AND value = task.impact) AS impact, ",
      "17":"(SELECT label FROM sys_choice where name = 'incident' AND element = 'urgency' AND value = task.urgency) AS urgency, ",
      "18":"(SELECT label FROM sys_choice where name = 'incident' AND element = 'severity' AND value = task.severity) AS severity, ",
      "19":"(SELECT user_name FROM sys_user WHERE sys_id = task.u_resolved_by) AS resolved_by, ",
      "20":"(SELECT name FROM sys_user where sys_id = task.u_resolved_by) AS resolved_by_name, ",
      "21":"(SELECT user_name FROM sys_user where sys_id = task.closed_by) AS closed_by, ",
      "22":"(SELECT name FROM sys_user where sys_id = task.closed_by) AS closed_by_name, ",
      "23":"(SELECT name FROM cmn_location WHERE sys_id = task.location) AS site ",
      "24":"FROM servicenow_dbname.task ",
      "25":"LEFT JOIN core_company ON task.company = core_company.sys_id ",
      "26":"LEFT JOIN cmdb_ci ON task.u_internal_service = cmdb_ci.sys_id ",
      "27":"LEFT JOIN cmn_department dept ON task.u_business_unit = dept.sys_id",
      "28":"LEFT JOIN task parent_task ON task.parent = parent_task.sys_id ",
      "29":"LEFT JOIN u_symptoms ON task.u_symptom = u_symptoms.sys_id ",
      "30":"WHERE task.sys_class_name = 'incident' "
    },
    "CoreFieldMapping": {
      "h_datelogged":"[logdate]",
      "h_dateresolved":"[u_resolved_at]",
      "h_dateclosed":"[closed_at]",
      "h_summary":"[short_description]",
      "h_description":"ServiceNow Incident Task Reference: [callref]\n\n[description]",
      "h_external_ref_number":"[callref]",
      "h_createdby":"[createdby_username]",
      "h_fk_user_id":"[incident_customer_username]",
      "h_fk_user_name":"[incident_customer_name]",
      "h_status":"[incident_state]",
      "h_request_language":"en-GB",
      "h_impact":"[impact]",
      "h_urgency":"[urgency]",
      "h_customer_type":"0",
      "h_container_id":"",
      "h_fk_serviceid":"ServiceNow Historic Requests",
      "h_resolution":"[close_notes]",
      "h_resolvedby_user_id":"[resolved_by]",
      "h_resolvedby_username":"[resolved_by_name]",
      "h_closedby_user_id":"[closed_by]",
      "h_closedby_username":"[closed_by_name]",
      "h_category_id":"",
      "h_category":"[symptom_name]",
      "h_closure_category_id":"",
      "h_closure_category":"[close_code]",
      "h_ownerid":"[owner_username]",
      "h_ownername":"[owner_name]",
      "h_fk_team_id":"[support_group]",
      "h_fk_priorityid":"[priority]",
      "h_site_id":"[site]",
      "h_source_type":"[contact_type]",
      "h_company_id":"",
      "h_company_name":"[company_name]",
      "h_withinfix":"[made_sla]",
      "h_withinresponse":"[made_sla]",
      "h_custom_a":"[request_guid]",
      "h_custom_b":"[service_name]",
      "h_custom_c":"[close_ci]",
      "h_custom_d":"[close_code]",
      "h_custom_e":"[createdby_username]",
      "h_custom_f":"[incident_customer_name]",
      "h_custom_g":"[owner_name]",
      "h_custom_h":"[department]",
      "h_custom_i":"[site]",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "AdditionalFieldMapping":{
      "h_firsttimefix":"",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":"",
      "h_flgproblemfix":"",
      "h_fk_problemfixid":"",
      "h_flgfixisworkaround":"",
      "h_flg_fixisresolution":""
    },
    "StatusMapping":{
      "ServiceNow Status":"Service Manager Status",
      "New":"status.new",
      "In Progress":"status.open",
      "Scheduled":"status.open",
      "Accepted":"status.open",
      "Resolved":"status.resolved",
      "Closed":"status.closed"
    },
    "PriorityMapping": {
      "ServiceNow Priority": "Service Manager Priority"
    },
    "ServiceMapping": {
      "ServiceNow Service Name":"Service Manager Service Name"
    }
  },
  "ConfServiceRequest": {
    "Import":false,
    "CallClass": "Service Request",
    "DefaultTeam":"Service Desk",
    "DefaultPriority":"Low",
    "DefaultService":"ServiceNow Historic Requests",
    "SQLStatement":{
      "0":"SELECT task.sys_id AS request_guid, task.sys_class_name AS callclass, task.number AS callref, ",
      "1":"task.made_sla, task.opened_at AS logdate, task.u_desk_visit, ",
      "2":"task.short_description, task.description, task.u_category, task.category_1, task.category_2, task.contact_type, ",
      "3":"task.u_resolved_at, task.closed_at, task.close_code, task.close_notes, ",
      "4":"task.u_total, task.u_total_string, task.u_unit_cost, task.u_ucost_string, task.u_quantity, task.u_payment_method, ",
      "5":"task.approval, task.u_it_approval_required, task.u_business_approval_required, task.sys_created_by AS createdby_username, ",
      "6":"dept.name as department, core_company.name AS company_name, cmdb_ci.name AS service_name, ",
      "7":"task.u_first_line_fix, parent_task.number AS parent_task_ref, wf.name AS workflow, ",
      "8":"(SELECT label FROM sys_choice where name = 'sc_request' AND element = 'request_state' AND value = task.request_state) AS request_state, ",
      "9":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state, ",
      "10":"(SELECT name from cmdb_ci where sys_id = task.u_close_ci) AS close_ci,",
      "11":"(SELECT user_name FROM sys_user where sys_id = task.opened_by) AS loggedby, ",
      "12":"(SELECT user_name FROM sys_user where sys_id = task.assigned_to) AS owner_username, ",
      "13":"(SELECT name FROM sys_user where sys_id = task.assigned_to) AS owner_name, ",
      "14":"(SELECT name FROM sys_user_group where sys_id = task.assignment_group) AS support_group, ",
      "15":"(SELECT label FROM sys_choice where name = 'task' AND element = 'priority' AND value = task.priority) AS priority, ",
      "16":"(SELECT label FROM sys_choice where name = 'task' AND element = 'impact' AND value = task.impact) AS impact, ",
      "17":"(SELECT label FROM sys_choice where name = 'task' AND element = 'urgency' AND value = task.urgency) AS urgency, ",
      "18":"(SELECT user_name FROM sys_user WHERE sys_id = task.requested_for) AS requested_for_username, ",
      "19":"(SELECT name FROM sys_user where sys_id = task.requested_for) AS requested_for_name, ",
      "20":"(SELECT user_name FROM sys_user WHERE sys_id = task.u_resolved_by) AS resolved_by, ",
      "21":"(SELECT name FROM sys_user where sys_id = task.u_resolved_by) AS resolved_by_name, ",
      "22":"(SELECT user_name FROM sys_user where sys_id = task.closed_by) AS closed_by, ",
      "23":"(SELECT name FROM sys_user where sys_id = task.closed_by) AS closed_by_name, ",
      "24":"(SELECT name FROM cmn_location WHERE sys_id = task.location) AS site ",
      "25":"FROM servicenow_dbname.task ",
      "26":"LEFT JOIN core_company ON task.company = core_company.sys_id ",
      "27":"LEFT JOIN cmdb_ci ON task.u_internal_service = cmdb_ci.sys_id ",
      "28":"LEFT JOIN cmn_department dept ON task.u_business_unit = dept.sys_id ",
      "29":"LEFT JOIN task parent_task ON task.parent = parent_task.sys_id ",
      "30":"LEFT JOIN wf_context wf ON task.sys_id = wf.id ",
      "31":"WHERE task.sys_class_name IN ('sc_request','sc_task')"
    },
    "CoreFieldMapping": {
      "h_datelogged":"[logdate]",
      "h_dateresolved":"[u_resolved_at]",
      "h_dateclosed":"[closed_at]",
      "h_summary":"[short_description]",
      "h_description":"ServiceNow Service Request Task Reference: [callref]\n\n[description]",
      "h_external_ref_number":"[callref]",
      "h_createdby":"[createdby_username]",
      "h_fk_user_id":"[requested_for_username]",
      "h_fk_user_name":"[requested_for_name]",
      "h_status":"[request_state]",
      "h_request_language":"en-GB",
      "h_impact":"[impact]",
      "h_urgency":"[urgency]",
      "h_customer_type":"0",
      "h_container_id":"",
      "h_fk_serviceid":"ServiceNow Historic Requests",
      "h_resolution":"[close_notes]",
      "h_resolvedby_user_id":"[resolved_by]",
      "h_resolvedby_username":"[resolved_by_name]",
      "h_closedby_user_id":"[closed_by]",
      "h_closedby_username":"[closed_by_name]",
      "h_category_id":"",
      "h_category":"[symptom_name]",
      "h_closure_category_id":"",
      "h_closure_category":"[close_code]",
      "h_ownerid":"[owner_username]",
      "h_ownername":"[owner_name]",
      "h_fk_team_id":"[support_group]",
      "h_fk_priorityid":"[priority]",
      "h_site_id":"[site]",
      "h_source_type":"[contact_type]",
      "h_company_id":"",
      "h_company_name":"[company_name]",
      "h_withinfix":"[made_sla]",
      "h_withinresponse":"[made_sla]",
      "h_custom_a":"[request_guid]",
      "h_custom_b":"[service_name]",
      "h_custom_c":"[close_ci]",
      "h_custom_d":"[close_code]",
      "h_custom_e":"[createdby_username]",
      "h_custom_f":"[requested_for_name]",
      "h_custom_g":"[owner_name]",
      "h_custom_h":"[department]",
      "h_custom_i":"[site]",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "AdditionalFieldMapping":{
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "StatusMapping":{
      "ServiceNow Status":"Service Manager Status",
      "New":"status.new",
      "In Progress":"status.open",
      "Scheduled":"status.open",
      "Accepted":"status.open",
      "Resolved":"status.resolved",
      "Closed Cancelled":"status.cancelled",
      "Closed":"status.closed"
    },
    "PriorityMapping": {
      "ServiceNow Priority": "Service Manager Priority"
    },
    "ServiceMapping": {
      "ServiceNow Service Name":"Service Manager Service Name"
    }
  },
  "ConfChangeRequest": {
    "Import":true,
    "CallClass": "Change Request",
    "DefaultTeam":"Service Desk",
    "DefaultPriority":"Low",
    "DefaultService":"ServiceNow Historic Requests",
    "SQLStatement":{
      "0":"SELECT task.sys_id AS request_guid, task.sys_class_name AS callclass, task.number AS callref, ",
      "1":"task.made_sla, task.opened_at AS logdate, task.u_desk_visit, task.type_1 AS change_type, ",
      "2":"task.short_description, task.description, task.u_category, task.category_1, task.category_2, task.contact_type, ",
      "3":"task.u_resolved_at, task.closed_at, task.u_closed_code, task.close_notes, task.approval, ",
      "4":"GROUP_CONCAT(DISTINCT dept.name) AS department, core_company.name AS company_name, cmdb_ci.name AS service_name, ",
      "5":"task.u_first_line_fix, parent_task.number AS parent_task_ref, wf.name AS workflow, task.u_justification, task.u_disruption, ",
      "6":"task.u_disruption_duration, task.backout_plan, task.u_support_plan, task.u_communication_plan, task.u_security_implication, task.change_plan, ",
      "7":"task.test_plan, task.u_implementation_result, task.u_imp_results, task.u_post_imp_results, ",
      "8":"(SELECT label FROM sys_choice where name = 'sc_request' AND element = 'request_state' AND value = task.request_state) AS request_state, ",
      "9":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state, ",
      "10":"(SELECT user_name FROM sys_user where sys_id = task.opened_by) AS loggedby, ",
      "11":"(SELECT user_name FROM sys_user where sys_id = task.assigned_to) AS owner_username, ",
      "12":"(SELECT name FROM sys_user_group where sys_id = task.assignment_group) AS support_group, ",
      "13":"(SELECT label FROM sys_choice where name = 'task' AND element = 'priority' AND value = task.priority) AS priority, ",
      "14":"(SELECT label FROM sys_choice where name = 'task' AND element = 'impact' AND value = task.impact) AS impact, ",
      "15":"(SELECT label FROM sys_choice where name = 'task' AND element = 'urgency' AND value = task.urgency) AS urgency, ",
      "16":"(SELECT user_name FROM sys_user WHERE sys_id = task.requested_for) AS requested_for_username, ",
      "17":"(SELECT name FROM sys_user where sys_id = task.requested_for) AS requested_for_name, ",
      "18":"(SELECT user_name FROM sys_user WHERE sys_id = task.u_resolved_by) AS resolved_by, ",
      "19":"(SELECT user_name FROM sys_user where sys_id = task.closed_by) AS closed_by, ",
      "20":"(SELECT name FROM sys_user where sys_id = task.closed_by) AS closed_by_name,",
      "21":"(SELECT name FROM cmn_location WHERE sys_id = task.location) AS site ",
      "22":"FROM servicenow_dbname.task ",
      "23":"LEFT JOIN core_company ON task.company = core_company.sys_id ",
      "24":"LEFT JOIN cmdb_ci ON task.u_service = cmdb_ci.sys_id ",
      "25":"LEFT JOIN cmn_department dept ON FIND_IN_SET(dept.sys_id, task.bu_affect) > 0 ",
      "26":"LEFT JOIN task parent_task ON task.parent = parent_task.sys_id ",
      "27":"LEFT JOIN wf_context wf ON task.sys_id = wf.id ",
      "28":"WHERE task.sys_class_name = 'change_request'",
      "29":"GROUP BY task.number"
    },
    "CoreFieldMapping": {
      "h_datelogged":"[logdate]",
      "h_dateresolved":"[u_resolved_at]",
      "h_dateclosed":"[closed_at]",
      "h_summary":"[short_description]",
      "h_description":"ServiceNow Change Request Task Reference: [callref]\n\n[description]\n\n\n\n'''Change Justification'''\n\n[u_justification]\n\n\n\n'''Change Disruption'''\n\n[u_disruption]\n\n'''Disruption Duration'''\n\n[u_disruption_duration]\n\n'''Security Implication'''\n\n[u_security_implication]\n\n'''Change Plan'''\n\n[change_plan]\n\n'''Backout Plan'''\n\n[backout_plan]\n\n'''Test Plan'''\n\n[test_plan]\n\n'''Support Plan'''\n\n[u_support_plan]\n\n'''Communication Plan'''\n\n[u_communication_plan]\n\n'''Approval Outcome'''\n\n[approval]\n\n'''Implementation result'''\n\n[u_implementation_result]\n\n'''Implementation results'''\n\n[u_imp_results]\n\n'''Post Implementation Results'''\n\n[u_post_imp_results]",
      "h_external_ref_number":"[callref]",
      "h_fk_user_id":"[requested_for_username]",
      "h_status":"[task_state]",
      "h_request_language":"en-GB",
      "h_impact":"[impact]",
      "h_urgency":"",
      "h_customer_type":"0",
      "h_container_id":"",
      "h_fk_serviceid":"ServiceNow Historic Requests",
      "h_resolution":"[close_notes]",
      "h_category_id":"[category_2]",
      "h_closure_category_id":"[u_closed_code]",
      "h_closedby_user_id":"[closed_by]",
      "h_closedby_username":"[closed_by_name]",
      "h_ownerid":"[owner_username]",
      "h_fk_team_id":"[support_group]",
      "h_fk_priorityid":"[priority]",
      "h_site_id":"[site]",
      "h_company_id":"",
      "h_company_name":"[company_name]",
      "h_withinfix":"[made_sla]",
      "h_withinresponse":"[made_sla]",
      "h_custom_a":"[request_guid]",
      "h_custom_b":"[service_name]",
      "h_custom_c":"",
      "h_custom_d":"[u_closed_code]",
      "h_custom_e":"",
      "h_custom_f":"[requested_for_name]",
      "h_custom_g":"",
      "h_custom_h":"[department]",
      "h_custom_i":"[site]",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "AdditionalFieldMapping":{
      "h_start_time":"",
      "h_end_time":"",
      "h_change_type":"[change_type]",
      "h_custom_a":"[category_2]",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":"",
      "h_scheduled":""
    },
    "StatusMapping":{
      "ServiceNow Status":"Service Manager Status",
      "Open":"status.open",
      "Pending":"status.open",
      "Closed Incomplete":"status.closed",
      "Closed Complete":"status.closed"
    },
    "PriorityMapping": {
      "ServiceNow Priority": "Service Manager Priority"
    },
    "ServiceMapping": {
      "ServiceNow Service Name":"Service Manager Service Name"
    }
  },
  "ConfProblem": {
    "Import":false,
    "CallClass": "Problem",
    "DefaultTeam":"Service Desk",
    "DefaultPriority":"Low",
    "DefaultService":"ServiceNow Historic Requests",
    "SQLStatement":{
      "0":"SELECT task.sys_id AS request_guid, task.sys_class_name AS callclass, task.number AS callref, ",
      "1":"task.made_sla, task.opened_at AS logdate, u_symptoms.u_name AS symptom_name, ",
      "2":"task.short_description, task.description, task.u_category, task.category_1, task.contact_type, ",
      "3":"task.u_resolved_at, task.closed_at, task.close_code, task.close_notes, ",
      "4":"core_company.name AS company_name, cmdb_ci.name AS service_name, wf.name AS workflow, ",
      "5":"task.u_first_line_fix, parent_task.number AS parent_task_ref, ",
      "6":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state, ",
      "7":"(SELECT user_name FROM sys_user where sys_id = task.opened_by) AS loggedby, ",
      "8":"(SELECT user_name FROM sys_user where sys_id = task.assigned_to) AS owner_username, ",
      "9":"(SELECT name FROM sys_user_group where sys_id = task.assignment_group) AS support_group, ",
      "10":"(SELECT label FROM sys_choice where name = 'task' AND element = 'priority' AND value = task.priority) AS priority, ",
      "11":"(SELECT user_name FROM sys_user WHERE sys_id = task.requested_for) AS requested_for_username, ",
      "12":"(SELECT user_name FROM sys_user WHERE sys_id = task.u_resolved_by) AS resolved_by, ",
      "13":"(SELECT user_name FROM sys_user where sys_id = task.closed_by) AS closed_by, ",
      "14":"(SELECT name FROM sys_user where sys_id = task.closed_by) AS closed_by_name,",
      "15":"(SELECT name FROM cmn_location WHERE sys_id = task.location) AS site ",
      "16":"FROM servicenow_dbname.task ",
      "17":"LEFT JOIN core_company ON task.company = core_company.sys_id ",
      "18":"LEFT JOIN cmdb_ci ON task.u_internal_service = cmdb_ci.sys_id",
      "19":"LEFT JOIN task parent_task ON task.parent = parent_task.sys_id",
      "20":"LEFT JOIN u_symptoms ON task.u_symptom = u_symptoms.sys_id",
      "21":"LEFT JOIN wf_context wf ON task.sys_id = wf.id ",
      "22":"WHERE task.sys_class_name = 'problem' AND task.known_error != 1"
    },
    "CoreFieldMapping": {
      "h_datelogged":"[logdate]",
      "h_dateresolved":"[u_resolved_at]",
      "h_dateclosed":"[closed_at]",
      "h_summary":"[short_description]",
      "h_description":"ServiceNow Problem Task Reference: [callref]\n\n[description]",
      "h_external_ref_number":"[callref]",
      "h_fk_user_id":"[requested_for_username]",
      "h_status":"[task_state]",
      "h_request_language":"en-GB",
      "h_impact":"",
      "h_urgency":"",
      "h_customer_type":"0",
      "h_container_id":"",
      "h_fk_serviceid":"ServiceNow Historic Requests",
      "h_resolution":"[close_notes]",
      "h_category_id":"[symptom_name]",
      "h_closure_category_id":"[close_code]",
      "h_closedby_user_id":"[closed_by]",
      "h_closedby_username":"[closed_by_name]",
      "h_ownerid":"[owner_username]",
      "h_fk_team_id":"[support_group]",
      "h_fk_priorityid":"[priority]",
      "h_site_id":"[site]",
      "h_company_id":"",
      "h_company_name":"[company_name]",
      "h_withinfix":"[made_sla]",
      "h_withinresponse":"[made_sla]",
      "h_custom_a":"[request_guid]",
      "h_custom_b":"[service_name]",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"[site]",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "AdditionalFieldMapping":{
      "h_workaround":"",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "StatusMapping":{
      "ServiceNow Status":"Service Manager Status",
      "Open":"status.open",
      "Closed Complete":"status.closed"
    },
    "PriorityMapping": {
      "ServiceNow Priority": "Service Manager Priority"
    },
    "ServiceMapping": {
      "ServiceNow Service Name":"Service Manager Service Name"
    }
  },
  "ConfKnownError": {
    "Import":false,
    "CallClass": "Known Error",
    "DefaultTeam":"Service Desk",
    "DefaultPriority":"Low",
    "DefaultService":"ServiceNow Historic Requests",
    "SQLStatement":{
      "0":"SELECT task.sys_id AS request_guid, task.sys_class_name AS callclass, task.number AS callref, ",
      "1":"task.made_sla, task.opened_at AS logdate, u_symptoms.u_name AS symptom_name, ",
      "2":"task.short_description, task.description, task.u_category, task.category_1, task.contact_type, ",
      "3":"task.u_resolved_at, task.closed_at, task.close_code, task.close_notes, ",
      "4":"core_company.name AS company_name, cmdb_ci.name AS service_name, wf.name AS workflow, ",
      "5":"task.u_first_line_fix, parent_task.number AS parent_task_ref, ",
      "6":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state, ",
      "7":"(SELECT user_name FROM sys_user where sys_id = task.opened_by) AS loggedby, ",
      "8":"(SELECT user_name FROM sys_user where sys_id = task.assigned_to) AS owner_username, ",
      "9":"(SELECT name FROM sys_user_group where sys_id = task.assignment_group) AS support_group, ",
      "10":"(SELECT label FROM sys_choice where name = 'task' AND element = 'priority' AND value = task.priority) AS priority, ",
      "11":"(SELECT user_name FROM sys_user WHERE sys_id = task.requested_for) AS requested_for_username, ",
      "12":"(SELECT user_name FROM sys_user WHERE sys_id = task.u_resolved_by) AS resolved_by, ",
      "13":"(SELECT user_name FROM sys_user where sys_id = task.closed_by) AS closed_by, ",
      "14":"(SELECT name FROM sys_user where sys_id = task.closed_by) AS closed_by_name,",
      "15":"(SELECT name FROM cmn_location WHERE sys_id = task.location) AS site ",
      "16":"FROM servicenow_dbname.task ",
      "17":"LEFT JOIN core_company ON task.company = core_company.sys_id ",
      "18":"LEFT JOIN cmdb_ci ON task.u_internal_service = cmdb_ci.sys_id",
      "19":"LEFT JOIN task parent_task ON task.parent = parent_task.sys_id",
      "20":"LEFT JOIN u_symptoms ON task.u_symptom = u_symptoms.sys_id",
      "21":"LEFT JOIN wf_context wf ON task.sys_id = wf.id ",
      "22":"WHERE task.sys_class_name = 'problem' AND task.known_error = 1"
    },
    "CoreFieldMapping": {
      "h_datelogged":"[logdate]",
      "h_dateresolved":"[u_resolved_at]",
      "h_dateclosed":"[closed_at]",
      "h_summary":"[short_description]",
      "h_description":"ServiceNow Known Error Task Reference: [callref]\n\n[description]",
      "h_external_ref_number":"[callref]",
      "h_fk_user_id":"[requested_for_username]",
      "h_status":"[task_state]",
      "h_request_language":"en-GB",
      "h_impact":"",
      "h_urgency":"",
      "h_customer_type":"0",
      "h_container_id":"",
      "h_fk_serviceid":"[service_name]",
      "h_resolution":"[close_notes]",
      "h_category_id":"[symptom_name]",
      "h_closure_category_id":"[close_code]",
      "h_ownerid":"[owner_username]",
      "h_fk_team_id":"[support_group]",
      "h_fk_priorityid":"[priority]",
      "h_site_id":"[site]",
      "h_company_id":"",
      "h_company_name":"[company_name]",
      "h_withinfix":"[made_sla]",
      "h_withinresponse":"[made_sla]",
      "h_custom_a":"[request_guid]",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"[site]",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "AdditionalFieldMapping":{
      "h_solution":"",
      "h_root_cause":"",
      "h_steps_to_resolve":"",
      "h_custom_a":"",
      "h_custom_b":"",
      "h_custom_c":"",
      "h_custom_d":"",
      "h_custom_e":"",
      "h_custom_f":"",
      "h_custom_g":"",
      "h_custom_h":"",
      "h_custom_i":"",
      "h_custom_j":"",
      "h_custom_k":"",
      "h_custom_l":"",
      "h_custom_m":"",
      "h_custom_n":"",
      "h_custom_o":"",
      "h_custom_p":"",
      "h_custom_q":""
    },
    "StatusMapping":{
      "ServiceNow Status":"Service Manager Status",
      "Open":"status.open",
      "Closed Complete":"status.closed"
    },
    "PriorityMapping": {
      "ServiceNow Priority": "Service Manager Priority"
    },
    "ServiceMapping": {
      "ServiceNow Service Name":"Service Manager Service Name"
    }
  },
  "ConfActivities": {
    "Import":false,
    "SQLStatement": {
      "0":"SELECT task.sys_class_name AS callclass, task.number AS callref, ",
      "1":"task.opened_at AS logdate, task.approval_set, parent_task.number AS parent_task_ref, ",
      "2":"sysappr.state AS approval_state, sysappr.u_rejection_reason, sysappr.expected_start, sysappr.due_date, ",
      "3":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state, ",
      "4":"(SELECT user_name FROM sys_user where sys_id = task.opened_by) AS loggedby, ",
      "5":"(SELECT name FROM sys_user_group where sys_id = task.assignment_group) AS support_group, ",
      "6":"(SELECT label FROM sys_choice where name = 'task' AND element = 'priority' AND value = task.priority) AS priority, ",
      "7":"(SELECT user_name FROM sys_user WHERE sys_id = task.requested_for) AS authoriser, ",
      "8":"(SELECT user_name FROM sys_user WHERE sys_id = task.closed_by) AS closed_by, ",
      "9":"(SELECT user_name FROM sys_user WHERE sys_id = sysappr.approver) AS approver, ",
      "10":"(SELECT name FROM wf_activity WHERE sys_id = sysappr.wf_activity) AS wf_activity, ",
      "11":"(SELECT label FROM sys_choice where name = 'task' AND element = 'state' AND value = task.state) AS task_state ",
      "12":"FROM servicenow_dbname.task ",
      "13":"LEFT JOIN task parent_task ON task.parent = parent_task.sys_id ",
      "14":"LEFT JOIN sysapproval_approver sysappr ON task.parent = sysappr.sysapproval ",
      "15":"WHERE task.sys_class_name = 'sysapproval_group' "
    },
    "Category":"BPM Authorisation",
    "ParentRef":"[parent_task_ref]",
    "Title":"Approval Activity",
    "Description":"Workflow: [wf_activity]\nPriority: [priority]\nRaised By: [loggedby]\nSupport Group: [support_group]",
    "StartDate":"[expected_start]",
    "DueDate":"[due_date]",
    "AssignTo":"[approver]",
    "Status":"[task_state]",
    "Decision":"[approval_state]",
    "Reason":"[u_rejection_reason]"
  },
  "TeamMapping": {
    "ServiceNow Team Name":"Service Manager Team Name"
  },
  "CategoryMapping": {
    "ServiceNow Category Name":"Service Manager Category ID"
  },
  "ResolutionCategoryMapping": {
    "ServiceNow Category Name":"Service Manager Category ID"
  }
}
```

#### HBConfig
Connection information for the Hornbill instance:
* "UserName" - Instance User Name with which the tool will log the new requests
* "Password" - Instance Password for the above User
* "InstanceId" - ID of your Hornbill instance

#### SNAppDBConf
Contains the connection information for the ServiceNow application database.
* "Driver" the driver to use to connect to the database that holds the ServiceNow application information:
    * mysql = MySQL Server v5.0 or above, or MariaDB
    * mssql = Microsoft SQL Server (2005 or above)
* "Server" The address of the SQL server
* "UserName" The username for the SQL database
* "Password" Password for above User Name
* "Port" SQL port
* "Encrypt" Boolean value to specify whether the connection between the script and the database should be encrypted. ''NOTE'': There is a bug in SQL Server 2008 and below that causes the connection to fail if the connection is encrypted. Only set this to true if your SQL Server has been patched accordingly.

#### CustomerType
Integer value 0 or 1, to determine the customer type for the records being imported:
* 0 - Hornbill Users
* 1 - Hornbill Contacts

#### ConfCallClass
Contains request-class specific configuration. This section should be repeated for all Service Manager Call Classes.
* Import - boolean true/false. Specifies whether the current class section should be included in the import.
* CallClass - specifies the Service Manager request class that the current Conf section relates to.
* DefaultTeam - If a request is being imported, and the tool cannot verify its Support Group, then the Support Group from this variable is used to assign the request.
* DefaultPriority - If a request is being imported, and the tool cannot verify its Priority, then the Priority from this variable is used to escalate the request.
* DefaultService - If a request is being imported, and the tool cannot verify its Service from the mapping, then the Service from this variable is used to log the request.
* SQLStatement - The SQL query used to get call (and extended) information from the ServiceNow application data. This is broken up in to numbered elements, for ease of reading and updating.
* CoreFieldMapping - The core fields used by the API calls to raise requests within Service Manager, and how the ServiceNow data should be mapped in to these fields.
    * Any value wrapped with [] will be populated with the corresponding response from the SQL Query
    * Any Other Value is treated literally as written example:
        * "h_summary":"[short_description]", - the value of short_description is taken from the SQL output and populated within the request summary field
        * "h_description":"ServiceNow Incident Task Reference: [callref]\n\n[description]", - the request description would be populated with "ServiceNow Incident Task Reference: ", followed by the ServiceNow Task reference, 2 new lines then the description text from the ServiceNow task.
    * Core Fields that can resolve associated record from passed-through value:   
        * "h_site_id":"[site]", - When a string is passed to the site field, the script attempts to resolve the given site name against the Site entity, and populates the request with the correct site information. If the site cannot be resolved, the site details are not populated for the request being imported.
        * "h_fk_user_id":"[requested_for_username]", - As site, above, but resolves the original request customer against the users or contacts within Hornbill. 
        * "h_ownerid":"[owner]", - As site, above, but resolves the original request owner against the analysts within Hornbill.
        * "h_category_id":"[symptom_name]", - As site, above, but uses additional CategoryMapping from the configuration, as detailed below.
        * "h_closure_category_id":"[close_code]", - As site, above, but uses additional ResolutionCategoryMapping from the configuration, as detailed below.
        * "h_ownerid":"[owner_username]", - As site, above, but resolves the original request owner against the analysts within Hornbill.
        * "h_fk_team_id":"[support_group]", - As site, above,  but uses additional TeamMapping from the configuration, as detailed below.
        * "h_fk_priorityid":"[priority]", - As site, above, but uses additional PriorityMapping from the configuration, as detailed below.
* AdditionalFieldMapping - Contains additional columns that can be stored against the new request record. Mapping rules are as above.
* StatusMapping - Allows for the mapping of task-class specific Statuses between ServiceNow and Hornbill Service Manager, where the left-side properties list the Statuses from ServiceNow, and the right-side values are the corresponding Statuses from Hornbill that should be used when importing requests.
* PriorityMapping - Allows for the mapping of task-class specific Priorities between ServiceNow and Hornbill Service Manager, where the left-side properties list the Priorities from ServiceNow, and the right-side values are the corresponding Priorities from Hornbill that should be used when escalating the imported requests.
* ServiceMapping - Allows for the mapping of task-class specific Services between ServiceNow and Hornbill Service Manager, where the left-side properties list the Service names from ServiceNow, and the right-side values are the corresponding Services from Hornbill that should be used when raising the new requests.

#### ConfActivities
Contains the configuration to allow the import of ServiceNow Approval Tasks as Hornbill Activities.
* Import - boolean true/false. Specifies whether Activities should be included in the import.
* SQLStatement - The SQL query used to get call (and extended) information from the ServiceNow application data. This is broken up in to numbered elements, for ease of reading and updating.
* Category - the Category of the Activity being raised within Hornbill (either `BPM Authorisation` or `Task`).
* ParentRef - The column containing the reference number of the parent task of the approval task being imported.
* Title - The summary title of the new Activities
* Description - The detailed description of the Activities
* StartDate - The Start Date of the Activities
* DueDate - The Due Date of the Activities
* AssignTo - The ID of the analyst that the Activity should be assigned to
* Status - The status of the Activity (Open, Closed Complete or Closed Incomplete)
* Decision - The approval decision
* Reason - The reason as to why the approval decision was made

#### TeamMapping
Allows for the mapping of Support Groups/Team between ServiceNow and Hornbill Service Manager, where the left-side properties list the Support Group names from ServiceNow, and the right-side values are the corresponding Team names from Hornbill that should be used when assigning the new requests.

#### CategoryMapping
Allows for the mapping of Problem Profiles/Request Categories between ServiceNow and Hornbill Service Manager, where the left-side properties list the Category label from ServiceNow, and the right-side values are the corresponding Profile Codes from Hornbill that should be used when categorising the new requests.

#### ResolutionCategoryMapping
Allows for the mapping of Resolution Profiles/Resolution Categories between ServiceNow and Hornbill Service Manager, where the left-side properties list the Resolution Category label from ServiceNow, and the right-side values are the corresponding Resolution Codes from Hornbill that should be used when applying Resolution Categories to the imported requests.

# Execute
Command Line Parameters
* file - Defaults to `conf.json` - Name of the Configuration file to load
* dryrun - Defaults to `false` - Set to true, and the XMLMC for new request creation will not be called and instead the XML will be dumped to the log file, this is to aid in debugging the initial connection information.
* debug - Defaults to `false` - Set this to true, and the log file will include additional debugging information. NOTE! The log file can increase in size dramatically with this flag set to true!
* zone - Defaults to `eur` - Allows you to change the ZONE used for creating the XMLMC EndPoint URL https://{ZONE}api.hornbill.com/{INSTANCE}/
* concurrent - defaults to `1`. This is to specify the number of requests that should be imported concurrently, and can be an integer between 1 and 10 (inclusive). 1 is the slowest level of import, but does not affect performance of your Hornbill instance, and 10 will process the import more quickly but may affect performance of your instance.
* attachments - defaults to `true`. By default, all attachments associated with the tasks that you import will be imported in to Service Manager and associated with the relevant requests. Set this to `false` to prevent any file attachments being imported.

# Testing
If you run the application with the argument dryrun=true then no requests will be logged - the XML used to raise requests will instead be saved in to the log file so you can ensure the data mappings are correct before running the import.

'servicenow_request_import_w64.exe -dryrun=true'

# Logging
All Logging output is saved in the log directory in the same directory as the executable the file name contains the date and time the import was run 'SN_Task_Import_2015-11-06T14-26-13Z.log'

# Error Codes
* `100` - Unable to create log File
* `101` - Unable to create log folder
* `102` - Unable to Load Configuration File
