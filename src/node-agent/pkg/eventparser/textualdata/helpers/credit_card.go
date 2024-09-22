package helpers

import (
	"strings"
	"unicode"
)

// List of card prefixes and their corresponding lengths
var cardPrefixes = map[string][]int{
	"34":       {15},     // AMEX
	"37":       {15},     // AMEX
	"300":      {15},     // Diners
	"301":      {15},     // Diners
	"302":      {15},     // Diners
	"303":      {15},     // Diners
	"36":       {15},     // Diners
	"38":       {15},     // Diners
	"6011":     {16},     // Discover
	"2014":     {16},     // Enroute
	"2149":     {16},     // Enroute
	"2100":     {16},     // JCB 15
	"1800":     {16},     // JCB 15
	"3088":     {16},     // JCB 16
	"3096":     {16},     // JCB 16
	"3112":     {16},     // JCB 16
	"3158":     {16},     // JCB 16
	"3337":     {16},     // JCB 16
	"3528":     {16},     // JCB 16
	"51":       {16},     // MasterCard
	"52":       {16},     // MasterCard
	"53":       {16},     // MasterCard
	"54":       {16},     // MasterCard
	"55":       {16},     // MasterCard
	"4":        {13, 16}, // Visa
	"4539":     {16},     // Visa
	"4556":     {16},     // Visa
	"4916":     {16},     // Visa
	"4532":     {16},     // Visa
	"4929":     {16},     // Visa
	"40240071": {16},     // Visa
	"4485":     {16},     // Visa
	"4716":     {16},     // Visa
	"8699":     {13, 16}, // Voyager
}

// CardRegex Matches the following card formats:
// 4111111111111111
// 4111 1111 1111 1111
// 4111-1111-1111-1111
// 4111.1111.1111.1111
// **** **** **** 1111
const CardRegex = `(?:(\d|\*)[ -\.]*?){13,19}`

// NormalizeCardNumber Normalize card number by removing non-digit characters (if separated by spaces, dots, or dashes)
func NormalizeCardNumber(card string) string {
	var normalized string
	for _, char := range card {
		if unicode.IsDigit(char) {
			normalized += string(char)
		}
	}
	return normalized
}

// IsValidCardNumber Check if the card number is valid based on prefixes and length
func IsValidCardNumber(card string) bool {
	for prefix, lengths := range cardPrefixes {
		if strings.HasPrefix(card, prefix) {
			for _, length := range lengths {
				if len(card) == length {
					return true
				}
			}
		}
	}
	return false
}
