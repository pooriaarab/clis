package model

import (
	"sort"
	"strings"
)

// Amount kinds from ParseCents. Blank and invalid never become zero;
// zero and negative stay out of priced distributions and are counted.
const (
	AmountBlank    = "blank"
	AmountPositive = "positive"
	AmountZero     = "zero"
	AmountNegative = "negative"
	AmountInvalid  = "invalid"
)

// ParseCents converts an amount cell to integer cents without float64.
// Only [+-]digits[.digits] with at most two fraction digits parses;
// anything else is invalid, never zero.
func ParseCents(s string) (int64, string) {
	t := strings.TrimSpace(s)
	if t == "" {
		return 0, AmountBlank
	}
	t, neg := strings.CutPrefix(t, "-")
	if !neg {
		t, _ = strings.CutPrefix(t, "+")
	}
	ip, fp, _ := strings.Cut(t, ".")
	if strings.Contains(fp, ".") {
		return 0, AmountInvalid
	}
	if ip == "" || len(fp) > 2 || !allDigits(ip) || !allDigits(fp) {
		return 0, AmountInvalid
	}
	// Pad missing fraction digits with zeros so the overflow check below
	// also guards the ×10 padding, not just the digits actually present.
	digits := ip + fp + strings.Repeat("0", 2-len(fp))
	var cents int64
	for _, d := range digits {
		if cents > (1<<62)/10 {
			return 0, AmountInvalid
		}
		cents = cents*10 + int64(d-'0')
	}
	if cents == 0 {
		return 0, AmountZero
	}
	if neg {
		return -cents, AmountNegative
	}
	return cents, AmountPositive
}

func allDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// CategoryShare is a supplier's share of its category total, in [0,1].
// A non-positive category total yields 0. Issue #55 consumes this.
func CategoryShare(supplierCents, categoryCents int64) float64 {
	if supplierCents <= 0 || categoryCents <= 0 {
		return 0
	}
	return float64(supplierCents) / float64(categoryCents)
}

type Tender struct {
	Reference       string   `json:"referenceNumber"`
	Solicitation    string   `json:"solicitationNumber"`
	Title           string   `json:"title"`
	TitleFR         string   `json:"titleFR"`
	Status          string   `json:"status"`
	Categories      []string `json:"categories"`
	UNSPSC          []string `json:"unspsc"`
	UNSPSCDesc      string   `json:"unspscDescription"`
	GSIN            []string `json:"gsin"`
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
		UNSPSC:          SplitSet(c("unspsc")),
		UNSPSCDesc:      c("unspscDescription-eng"),
		GSIN:            SplitSet(c("gsin-nibs")),
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

// Award is one award notice. AmountCents holds contractAmount in cents;
// AmountKind is blank, positive, zero, negative or invalid.
// UNSPSC and GSIN are []string because those cells hold newline-separated
// values, the same shape as Tender.
type Award struct {
	Reference    string   `json:"referenceNumber"`
	Solicitation string   `json:"solicitationNumber"`
	AwardDate    string   `json:"contractAwardDate"`
	AmountRaw    string   `json:"contractAmount"`
	AmountCents  int64    `json:"contractAmountCents"`
	AmountKind   string   `json:"contractAmountKind"`
	Currency     string   `json:"contractCurrency"`
	GSIN         []string `json:"gsin"`
	UNSPSC       []string `json:"unspsc"`
	Categories   []string `json:"categories"`
	Supplier     string   `json:"supplier"`
	Org          string   `json:"contractingEntity"`
}

// ParseAward copies each cell. ReuseRecord invalidates the row on Next.
func ParseAward(get func(string) string) Award {
	c := func(name string) string { return strings.Clone(get(name)) }
	raw := c("contractAmount-montantContrat")
	cents, kind := ParseCents(raw)
	return Award{
		Reference:    c("referenceNumber-numeroReference"),
		Solicitation: c("solicitationNumber-numeroSollicitation"),
		AwardDate:    c("contractAwardDate-dateAttributionContrat"),
		AmountRaw:    raw,
		AmountCents:  cents,
		AmountKind:   kind,
		Currency:     c("contractCurrency-contratMonnaie"),
		GSIN:         SplitSet(c("gsin-nibs")),
		UNSPSC:       SplitSet(c("unspsc")),
		Categories:   SplitSet(c("procurementCategory-categorieApprovisionnement")),
		Supplier:     c("supplierLegalName-nomLegalFournisseur-eng"),
		Org:          c("contractingEntityName-nomEntitContractante-eng"),
	}
}

// Contract is one contract-history row: an award plus its record count.
type Contract struct {
	Award
	Records string `json:"numberOfRecords"`
}

// ParseContract copies each cell. ReuseRecord invalidates the row on Next.
func ParseContract(get func(string) string) Contract {
	return Contract{
		Award:   ParseAward(get),
		Records: strings.Clone(get("numberOfRecords-nombreEnregistrements")),
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
