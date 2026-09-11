package model

import (
	"sort"
	"strings"
)

type Tender struct {
	Reference       string   `json:"referenceNumber"`
	Solicitation    string   `json:"solicitationNumber"`
	Title           string   `json:"title"`
	TitleFR         string   `json:"titleFR"`
	Status          string   `json:"status"`
	Categories      []string `json:"categories"`
	UNSPSC          string   `json:"unspsc"`
	UNSPSCDesc      string   `json:"unspscDescription"`
	GSIN            string   `json:"gsin"`
	GSINDesc        string   `json:"gsinDescription"`
	NoticeType      string   `json:"noticeType"`
	Method          string   `json:"procurementMethod"`
	Org             string   `json:"contractingEntity"`
	EndUser         string   `json:"endUserEntities"`
	Regions         []string `json:"regionsOfDelivery"`
	TradeAgreements []string `json:"tradeAgreements"`
	Publication     string   `json:"publicationDate"`
	Closing         string   `json:"tenderClosingDate"`
	ContractStart   string   `json:"expectedContractStartDate"`
	Description     string   `json:"description"`
	DescriptionFR   string   `json:"descriptionFR"`
}

// Parse copies each cell. ReuseRecord invalidates the current row on Next.
func Parse(get func(string) string) Tender {
	c := func(name string) string { return strings.Clone(get(name)) }
	return Tender{
		Reference:       c("referenceNumber-numeroReference"),
		Solicitation:    c("solicitationNumber-numeroSollicitation"),
		Title:           c("title-titre-eng"),
		TitleFR:         c("title-titre-fra"),
		Status:          c("tenderStatus-appelOffresStatut-eng"),
		Categories:      SplitSet(c("procurementCategory-categorieApprovisionnement")),
		UNSPSC:          c("unspsc"),
		UNSPSCDesc:      c("unspscDescription-eng"),
		GSIN:            c("gsin-nibs"),
		GSINDesc:        c("gsinDescription-nibsDescription-eng"),
		NoticeType:      c("noticeType-avisType-eng"),
		Method:          c("procurementMethod-methodeApprovisionnement-eng"),
		Org:             c("contractingEntityName-nomEntitContractante-eng"),
		EndUser:         c("endUserEntitiesName-nomEntitesUtilisateurFinal-eng"),
		Regions:         SplitSet(c("regionsOfDelivery-regionsLivraison-eng")),
		TradeAgreements: SplitSet(c("tradeAgreements-accordsCommerciaux-eng")),
		Publication:     c("publicationDate-datePublication"),
		Closing:         c("tenderClosingDate-appelOffresDateCloture"),
		ContractStart:   c("expectedContractStartDate-dateDebutContratPrevue"),
		Description:     c("tenderDescription-descriptionAppelOffres-eng"),
		DescriptionFR:   c("tenderDescription-descriptionAppelOffres-fra"),
	}
}

// SplitSet parses a newline-separated multi-value cell into a sorted set:
// "*GD\n*SRV" and "*SRV\n*GD" both yield ["*GD" "*SRV"].
func SplitSet(s string) []string {
	var out []string
	seen := map[string]bool{}
	for _, p := range strings.Split(s, "\n") {
		if p = strings.TrimSpace(p); p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
