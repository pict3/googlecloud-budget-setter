package main

import (
	"net/http"
	log "github.com/sirupsen/logrus"
	"context"

	"io/ioutil"
	"encoding/json"
	"time"

	"strings"
	"strconv"

	budgets "cloud.google.com/go/billing/budgets/apiv1"
	budgetspb "cloud.google.com/go/billing/budgets/apiv1/budgetspb"
	"google.golang.org/genproto/googleapis/type/money"
)


type BacklogIssueInfo struct {
	ID      int `json:"id"`
	Project struct {
		ID                                int    `json:"id"`
		ProjectKey                        string `json:"projectKey"`
		Name                              string `json:"name"`
		ChartEnabled                      bool   `json:"chartEnabled"`
		SubtaskingEnabled                 bool   `json:"subtaskingEnabled"`
		ProjectLeaderCanEditProjectLeader bool   `json:"projectLeaderCanEditProjectLeader"`
		UseWikiTreeView                   bool   `json:"useWikiTreeView"`
		TextFormattingRule                string `json:"textFormattingRule"`
		Archived                          bool   `json:"archived"`
	} `json:"project"`
	Type    int `json:"type"`
	Content struct {
		ID          int    `json:"id"`
		KeyID       int    `json:"key_id"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		IssueType   struct {
			ID           int    `json:"id"`
			ProjectID    int    `json:"projectId"`
			Name         string `json:"name"`
			Color        string `json:"color"`
			DisplayOrder int    `json:"displayOrder"`
		} `json:"issueType"`
		Resolution interface{} `json:"resolution"`
		Priority   struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"priority"`
		Status struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"status"`
		Assignee       interface{}   `json:"assignee"`
		Category       []interface{} `json:"category"`
		Versions       []interface{} `json:"versions"`
		Milestone      []interface{} `json:"milestone"`
		StartDate      interface{}   `json:"startDate"`
		DueDate        interface{}   `json:"dueDate"`
		EstimatedHours interface{}   `json:"estimatedHours"`
		ActualHours    interface{}   `json:"actualHours"`
		ParentIssueID  interface{}   `json:"parentIssueId"`
		CustomFields   []interface{} `json:"customFields"`
		Attachments    []interface{} `json:"attachments"`
	} `json:"content"`
	Notifications []interface{} `json:"notifications"`
	CreatedUser   struct {
		ID           int         `json:"id"`
		UserID       interface{} `json:"userId"`
		Name         string      `json:"name"`
		RoleType     int         `json:"roleType"`
		Lang         interface{} `json:"lang"`
		MailAddress  interface{} `json:"mailAddress"`
		NulabAccount interface{} `json:"nulabAccount"`
	} `json:"createdUser"`
	Created time.Time `json:"created"`
}

type DescriptionInfo struct {
	projectId         string
	billingAccountId  string
	budget            int64
}


func handler(writer http.ResponseWriter, request *http.Request) {
	log.Info("handler called")

	userAgent := request.Header["User-Agent"]
	if userAgent[0] != "Backlog Webhook" {
		log.WithFields(log.Fields{
			"userAgent": userAgent,
		}).Error("Invalid userAgent.")
	
		return
	} 

	switch request.Method {
	case "POST":
		body, _ := ioutil.ReadAll(request.Body)

		var backlogIssueInfo BacklogIssueInfo

		if err := json.Unmarshal([]byte(body), &backlogIssueInfo); err != nil {
			log.WithFields(log.Fields{
				"error-msg": err,
			}).Error("Unmarshal()")

			return
		}

		description := backlogIssueInfo.Content.Description
		descriptionInfo := getDescriptionInfo(description)

		if ret := createBudget(descriptionInfo); ret != true {
			log.Error("createBudget Failed.")

			return
		}

	default:
		log.Error("Invalid method.")

		return
	}

	log.Info("Create budjet success.")
}

func getDescriptionInfo(description string) DescriptionInfo {
	log.WithFields(log.Fields{
		"description": description,
	}).Info("getDescriptionInfo start.")

	projectIdStr :=  "ProjectId: "
	projectIdIdx := strings.Index(description, projectIdStr)

	BillingAccountIdStr :=  "BillingAccountId: "
	BillingAccountIdIdx := strings.Index(description, BillingAccountIdStr)

	budgetStr :=  "Budget[¥]: "
	budgetIdx := strings.Index(description, budgetStr)

	projectId := description[projectIdIdx + len(projectIdStr):BillingAccountIdIdx - 1]
	billingAccountId := description[BillingAccountIdIdx + len(BillingAccountIdStr):budgetIdx - 1]
	budget, _ := strconv.Atoi(description[budgetIdx + len(budgetStr):len(description)])

	return DescriptionInfo{
		projectId:        projectId,
		billingAccountId: billingAccountId,
		budget:           int64(budget), 
	}
}

func createBudget(descriptionInfo DescriptionInfo) bool{
	log.WithFields(log.Fields{
		"descriptionInfo": descriptionInfo,
	}).Info("createBudget start.")

	ctx := context.Background()

	c, err := budgets.NewBudgetClient(ctx)
	if err != nil {
		log.WithFields(log.Fields{
			"error-msg": err,
		}).Error("NewBudgetClient")

		return false
	}
	defer c.Close()

	budget := &budgetspb.Budget {
		DisplayName: descriptionInfo.projectId,
		BudgetFilter: &budgetspb.Filter {
			CreditTypesTreatment: budgetspb.Filter_INCLUDE_ALL_CREDITS, 
			UsagePeriod: &budgetspb.Filter_CalendarPeriod {
				CalendarPeriod: budgetspb.CalendarPeriod_MONTH,
			},
		},
		Amount: &budgetspb.BudgetAmount {
			BudgetAmount: &budgetspb.BudgetAmount_SpecifiedAmount {
				SpecifiedAmount: &money.Money {
					CurrencyCode: "JPY",
					Units:        descriptionInfo.budget,
				},
			},
		},
		NotificationsRule: &budgetspb.NotificationsRule {
			DisableDefaultIamRecipients: true,
		},
	}

	
	req := &budgetspb.CreateBudgetRequest{
		Parent: `billingAccounts/` + descriptionInfo.billingAccountId,
		Budget: budget,
	}

	resp, err := c.CreateBudget(ctx, req)
	if err != nil {
		log.WithFields(log.Fields{
			"error-msg": err,
			"req": req,
		}).Error("CreateBudget")

		return false
	}

	log.WithFields(log.Fields{
		"resp": resp,
	}).Info("CreateBudget")

	return true
}


func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
