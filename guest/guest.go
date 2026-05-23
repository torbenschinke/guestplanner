package guest

type ID string
type Guest struct {
	ID               ID `visible:"false"`
	Name             string
	Single           bool
	Salutation       string `label:"Anrede"`
	RegistrationCode string `label:"Anmeldecode" supportingText:"Beim erstmaligen Erstellen Feld leerlassen und der Code wird automatisch vergeben."`
	Adults           int    `value:"\"1\"" label:"Anzahl Erwachsene"`
	Children         int    `value:"\"0\"" label:"Anzahl Kinder"`
	Status           Status `values:"[\"wartet\",\"angenommen\",\"abgesagt\"]"`
	Invited          bool   `label:"Einladung (SMS) versendet"`
}

func (g Guest) WithIdentity(id ID) Guest {
	g.ID = id
	return g
}

func (g Guest) Identity() ID {
	return g.ID
}

type Status string

const (
	StatusWaiting   Status = "wartet"
	StatusConfirmed Status = "angenommen"
	StatusDeclined  Status = "abgesagt"
)
