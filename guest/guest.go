package guest

type ID string
type Guest struct {
	ID               ID `visible:"false"`
	Name             string
	Single           bool
	Salutation       string
	RegistrationCode string
	Adults           int `value:"\"1\""`
	Children         int
	Status           Status `values:"[\"wartet\",\"angenommen\",\"abgesagt\"]"`
	Invited          bool
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
