package psbotfunc

type Food struct {
	Name   string `json:"name"`
	Num    int    `json:"num"`
	Energy int    `json:"energy"`
}

type Cook struct {
	Name   string  `json:"name"`
	Recipe []*Food `json:"recipe"`
}
