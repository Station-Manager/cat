package cmd

type CatCmd string

func (cc CatCmd) String() string {
	return string(cc)
}

const (
	Init     CatCmd = "INIT"
	Read     CatCmd = "READ"
	PlayBack CatCmd = "PLAYBACK"
)
