package recordproof

type Item struct {
	Name    string
	Enabled bool
}

func Label(name string) string {
	item := Item{Name: name, Enabled: true}
	return item.Name
}
