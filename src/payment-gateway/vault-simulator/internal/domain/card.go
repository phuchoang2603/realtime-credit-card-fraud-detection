package domain

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
)

type Behavior string

const (
	BehaviorOK      Behavior = "ok"
	BehaviorNSF     Behavior = "nsf"
	BehaviorIssuer  Behavior = "issuer"
	BehaviorInvalid Behavior = "invalid"
)

const (
	MethodTypeCard     = "card"
	serialsPerBehavior = 2000
	cardNumberLength   = 16
	firstGeneratedRef  = 5
)

type Card struct {
	Ref      string
	Behavior Behavior
}

var namedCards = map[string]Card{
	"4242424242424242": {Ref: "card-0001", Behavior: BehaviorOK},
	"4000000000009995": {Ref: "card-0002", Behavior: BehaviorNSF},
	"4000000000000002": {Ref: "card-0003", Behavior: BehaviorIssuer},
	"4000000000000069": {Ref: "card-0004", Behavior: BehaviorInvalid},
}

var generatedRanges = []struct {
	prefix   string
	behavior Behavior
}{
	{"4111", BehaviorOK},
	{"4112", BehaviorNSF},
	{"4113", BehaviorIssuer},
	{"4114", BehaviorInvalid},
}

func LookupCard(number string) (Card, bool) {
	if card, ok := namedCards[number]; ok {
		return card, true
	}
	if len(number) != cardNumberLength || !allDigits(number) || !LuhnValid(number) {
		return Card{}, false
	}
	for index, r := range generatedRanges {
		if number[:4] != r.prefix {
			continue
		}
		serial, err := strconv.Atoi(number[4 : cardNumberLength-1])
		if err != nil || serial >= serialsPerBehavior {
			return Card{}, false
		}
		return Card{
			Ref:      fmt.Sprintf("card-%04d", firstGeneratedRef+index*serialsPerBehavior+serial),
			Behavior: r.behavior,
		}, true
	}
	return Card{}, false
}

func LuhnValid(number string) bool {
	if number == "" || !allDigits(number) {
		return false
	}
	sum := 0
	double := false
	for i := len(number) - 1; i >= 0; i-- {
		digit := int(number[i] - '0')
		if double {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		double = !double
	}
	return sum%10 == 0
}

func Fingerprint(key []byte, cardRef string) string {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(cardRef))
	return hex.EncodeToString(mac.Sum(nil))
}

func allDigits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return true
}
