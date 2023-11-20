package budget

import (
	"testing"
	"time"
)

func TestSMSTimestampRE(t *testing.T) {
	t.Run("timestamp", func(t *testing.T) {
		var sms = `1.00 GEL
MC WORLD ELITE (***3122)
LTD MP DEVELOPMENT 20/11/2023 22:30:50
Balance: 1072.80 GEL`

		match := smsTimestampRE.FindString(sms)

		if match != "20/11/2023 22:30:50" {
			t.Fail()
		}

		timestamp, err := time.Parse(smsTimestampFormat, match)
		if err != nil {
			t.Error(err)
		}

		if timestamp.UTC().Unix() != 1700519450 {
			t.Error("timestamp.UTC().Unix() != 1700519450")
		}
	})

	t.Run("no timestamp", func(t *testing.T) {
		var sms = `1.00 GEL
MC WORLD ELITE (***3122)
Balance: 1072.80 GEL`
		match := smsTimestampRE.FindString(sms)

		if match != "" {
			t.Error(match)
		}
	})
}
