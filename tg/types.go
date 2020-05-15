package tg

// UserMsg is a plain user text message
type UserMsg struct {
	ID   int
	Text string
}

// Btn is a telegram inline btn
type Btn struct {
	ID   string
	Text string
}

// BtnClick is a telegram inline btn reply
type BtnClick struct {
	MessageID int
	BtnID     string
}

type BotMessage struct {
	ReplyToMsgID int
	Text         string
	TextMarkdown bool
	Btns         []Btn
}
