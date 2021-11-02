package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/olivere/elastic/v7"
	"github.com/rustingoff/excel/entity"
	"github.com/xuri/excelize/v2"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Handler interface {
	Home(c *gin.Context)
	Campaign(c *gin.Context)
	Show(c *gin.Context)
	Export(c *gin.Context)
	Delete(c *gin.Context)
}

type handler struct {
	elastic *elastic.Client
}

func GetHandlerPackage(elastic *elastic.Client) Handler {
	indc := elastic.CreateIndex("amazon_campaign")

	if indc == nil {
		log.Println("can't create index")
	}

	return &handler{elastic}
}

func (h *handler) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", nil)
}

func (h *handler) Campaign(c *gin.Context) {
	if c.Request.Method == http.MethodGet {
		c.HTML(http.StatusOK, "post.html", nil)
		return
	}

	var totalKeywords uint64
	today, _ := time.LoadLocation("America/Los_Angeles")
	keywords := strings.Split(c.PostForm("keywords"), "\r\n")
	negativeKeywords := strings.Split(c.PostForm("negative_keywords"), "\r\n")
	bid, _ := strconv.ParseFloat(c.PostForm("bid"), 32)
	if c.PostForm("total_keywords") == "" {
		totalKeywords = uint64(len(keywords))
	} else {
		totalKeywords, _ = strconv.ParseUint(c.PostForm("total_keywords"), 10, 32)
	}
	dailyBudget, _ := strconv.ParseFloat(c.PostForm("daily_budget"), 32)

	model := entity.CampaignEntity{
		CampaignName:      c.PostForm("campaign_name"),
		CampaignStartDate: time.Now().In(today).Format("01/02/2006"),
		DailyBudget:       fmt.Sprintf("%.2f", dailyBudget),
		MatchType:         c.PostForm("match_type"),
		Bid:               fmt.Sprintf("%.2f", bid),
		SKU:               c.PostForm("sku"),
		TotalKeywords:     uint(totalKeywords),
		Keywords:          keywords,
		NegativeMatchType: c.PostForm("negative_match_type"),
		NegativeKeywords:  negativeKeywords,
	}

	_, err := h.elastic.Index().Index("amazon_campaign").BodyJson(&model).Do(context.TODO())
	if err != nil {
		fmt.Println(err.Error())
	}

	c.Redirect(http.StatusMovedPermanently, "/campaign")
}

func (h *handler) Show(c *gin.Context) {
	campaigns, err := getCampaigns(h.elastic)
	if err != nil {
		log.Println(err.Error())
		c.Redirect(http.StatusMovedPermanently, "/")
		return
	}

	c.HTML(http.StatusOK, "show.html", campaigns)
}

func (h *handler) Export(c *gin.Context) {
	campaignID := c.Param("id")

	campaigns, err := getCampaign(h.elastic, campaignID)
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/")
	}

	f, err := excelize.OpenFile("static/template.xlsx")
	if err != nil {
		fmt.Println(err)
		return
	}

	var matchType string
	campaignsCount := len(campaigns.Keywords) / int(campaigns.TotalKeywords)
	count := 0
	restKeyCount := 0

	for j := 0; j < campaignsCount; j++ {
		if j > 0 {
			matchType = " - " + campaigns.MatchType + fmt.Sprint(j)
		} else {
			matchType = " - " + campaigns.MatchType
		}
		c, err := writeExportCampaign(f, (j*6)+count, campaigns, matchType, campaigns.Keywords[int(campaigns.TotalKeywords)*j:int(campaigns.TotalKeywords)*(j+1)])
		if err != nil {
			panic(err)
		}
		count += int(campaigns.TotalKeywords) + c
		if j == campaignsCount-1 && campaignsCount > 1 {
			restKeyCount = len(campaigns.Keywords) % int(campaigns.TotalKeywords)
			if restKeyCount > 0 {
				_, err = writeExportCampaign(f, ((j+1)*6)+count, campaigns, " - Exact1", campaigns.Keywords[int(campaigns.TotalKeywords)*(j+1):len(campaigns.Keywords)])
				if err != nil {
					panic(err)
				}
			}
			count += restKeyCount + 6
		}
	}

	err = f.SaveAs("static/exports/" + campaigns.ID + ".xlsx")
	if err != nil {
		log.Println(err.Error())
		panic(err)
	}

	c.Redirect(http.StatusMovedPermanently, "https://multiply.today/static/"+campaigns.ID+".xlsx")
}

func (h *handler) Delete(c *gin.Context) {
	campaignID := c.Param("id")

	_, err := h.elastic.Delete().Index("amazon_campaign").Id(campaignID).Do(context.TODO())
	if err != nil {
		panic(err)
	}

	time.Sleep(time.Millisecond * 1000)
	c.Redirect(http.StatusMovedPermanently, "/show")
}

func getCampaigns(client *elastic.Client) ([]entity.CampaignEntity, error) {
	res, err := client.Search("amazon_campaign").Do(context.TODO())
	if err != nil {
		return nil, err
	}

	var campaigns = make([]entity.CampaignEntity, 0)
	for i := 0; i < int(res.TotalHits()); i++ {
		var campaign entity.CampaignEntity
		campaignByte := res.Hits.Hits[i].Source
		_ = json.Unmarshal(campaignByte, &campaign)
		campaign.ID = res.Hits.Hits[i].Id
		campaigns = append(campaigns, campaign)
	}

	return campaigns, nil
}

func writeExportCampaign(f *excelize.File, count int, campaign entity.CampaignEntity, nameExact string, keywords []string) (int, error) {
	err := f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", count+2), "Campaign")
	if err != nil {
		return 0, err
	}

	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", count+2), campaign.CampaignName+nameExact)
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("E%d", count+2), campaign.DailyBudget)
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("G%d", count+2), campaign.CampaignStartDate)
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("I%d", count+2), "Manual")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("P%d", count+2), "enabled")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("Z%d", count+2), "Dynamic bidding (down only)")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("AA%d", count+2), "All")
	if err != nil {
		return 0, err
	}

	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("AA%d", count+3), "Top of search (page 1)")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("AB%d", count+3), "0%")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", count+3), "Campaign By Placement")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", count+3), campaign.CampaignName+nameExact)
	if err != nil {
		return 0, err
	}

	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("AA%d", count+4), "Rest of search")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", count+4), "Campaign By Placement")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", count+4), campaign.CampaignName+nameExact)
	if err != nil {
		return 0, err
	}

	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("AA%d", count+5), "Product pages")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("AB%d", count+5), "0%")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", count+5), "Campaign By Placement")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", count+5), campaign.CampaignName+nameExact)
	if err != nil {
		return 0, err
	}

	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", count+6), "Ad Group")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("K%d", count+6), campaign.Bid)
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("P%d", count+6), "enabled")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("Q%d", count+6), "enabled")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("J%d", count+6), "Ad Group 1")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", count+6), campaign.CampaignName+nameExact)
	if err != nil {
		return 0, err
	}

	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", count+7), "Ad")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("J%d", count+7), "Ad Group 1")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", count+7), campaign.CampaignName+nameExact)
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("O%d", count+7), campaign.SKU)
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("P%d", count+7), "enabled")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("Q%d", count+7), "enabled")
	if err != nil {
		return 0, err
	}
	err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("R%d", count+7), "enabled")
	if err != nil {
		return 0, err
	}

	for j, keyword := range keywords {
		err := f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", j+8+count), "Keyword")
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("J%d", j+8+count), "Ad Group 1")
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", j+8+count), campaign.CampaignName+nameExact)
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("L%d", j+8+count), keyword)
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("P%d", j+8+count), "enabled")
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("Q%d", j+8+count), "enabled")
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("R%d", j+8+count), "enabled")
		if err != nil {
			return 0, err
		}
		err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("N%d", j+8+count), campaign.MatchType)
		if err != nil {
			return 0, err
		}

		if campaign.NegativeMatchType == "campaign negative phrase" || campaign.NegativeMatchType == "campaign negative exact" {
			for k, negativeKeyword := range campaign.NegativeKeywords {
				err := f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("B%d", j+k+9+count), "Keyword")
				if err != nil {
					return 0, err
				}
				err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("D%d", j+k+9+count), campaign.CampaignName+nameExact)
				if err != nil {
					return 0, err
				}
				err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("L%d", j+k+9+count), negativeKeyword)
				if err != nil {
					return 0, err
				}
				err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("N%d", j+k+9+count), campaign.NegativeMatchType)
				if err != nil {
					return 0, err
				}
				err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("P%d", j+k+9+count), "enabled")
				if err != nil {
					return 0, err
				}
				err = f.SetCellValue("Sponsored Products Campaigns", fmt.Sprintf("R%d", j+k+9+count), "enabled")
				if err != nil {
					return 0, err
				}
			}
		}
	}
	return len(campaign.NegativeKeywords), nil
}

func getCampaign(client *elastic.Client, campaignID string) (entity.CampaignEntity, error) {
	query := elastic.NewMatchQuery("_id", campaignID)

	res, err := client.Search("amazon_campaign").Query(query).Do(context.TODO())
	if err != nil {
		return entity.CampaignEntity{}, err
	}

	var campaign entity.CampaignEntity
	campaignByte := res.Hits.Hits[0].Source

	err = json.Unmarshal(campaignByte, &campaign)
	if err != nil {
		return entity.CampaignEntity{}, err
	}

	campaign.ID = res.Hits.Hits[0].Id

	return campaign, nil
}
