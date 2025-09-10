package domain

type EventSet struct {
	Namespace string
	Kind      string
	Events    []Event
}

type Event struct {
	Kind    string
	Message string
	Reason  string
}
