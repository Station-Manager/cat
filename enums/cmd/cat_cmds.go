package cmd

type CatCmd string

func (cc CatCmd) String() string {
	return string(cc)
}
