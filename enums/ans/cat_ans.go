package ans

type CatAns string

func (ca CatAns) String() string {
	return string(ca)
}

const (
	Identity CatAns = "IDENTITY"
	VfoAFreq CatAns = "VFOAFREQ"
	VfoBFreq CatAns = "VFOBFREQ"
	Split    CatAns = "SPLIT"
	Select   CatAns = "SELECT"
	MainMode CatAns = "MAINMODE"
	SubMode  CatAns = "SUBMODE"
	TxPwr    CatAns = "TXPWR"
)
