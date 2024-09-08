
#include "pci.h"

// Check if the character is a digit
static __inline bool is_digit(char c) {
    return c >= '0' && c <= '9';
}

// Luhn check to validate a card number
// Ref: https://en.wikipedia.org/wiki/Luhn_algorithm
bool luhn_check(const char *card_number, int len) {
    __u32 sum = 0;
    bool even = true;  // Start with alternating since the check digit is not doubled

    // Process all digits except the last one (check digit)
    for (int i = len - 2; i >= 0; i--) {
        __u32 digit = card_number[i] - '0';  // Convert character to integer
        if (digit < 0 || digit > 9) return false;  // value may only contain digits
        if (even) digit *= 2; // double the value
        if (digit > 9) digit -= 9;

        even = !even;
        sum += digit;
    }

    // Add the check digit (last digit) to the sum
    __u32 checksum = card_number[len - 1] - '0';
    sum += checksum;

    // If the total modulo 10 is 0, the number is valid
    return (sum % 10 == 0);
}

// Helper function to check for card-like sequences
bool detect_card_number(const char *data, int data_len) {
    int digit_count = 0;
    char card_number[MAX_CARD_LEN];  // Store potential card number sequence

    for (int i = 0; i < data_len; i++) {
        if (is_digit(data[i])) {
            card_number[digit_count] = data[i];  // Store digit
            digit_count++;

            // Reset count if we exceed the maximum card number length
            if (digit_count > MAX_CARD_LEN) digit_count = 0;
        } else {
            // Check if we have a valid card number length and validate with Luhn check
            if (digit_count >= MIN_CARD_LEN && digit_count <= MAX_CARD_LEN) {
                if (luhn_check(card_number, digit_count)) return true;
            }

            digit_count = 0;  // Reset count if non-digit found
        }
    }

    // Check again at the end in case the card number is at the end of the string
    if (digit_count >= MIN_CARD_LEN && digit_count <= MAX_CARD_LEN) {
        return luhn_check(card_number, digit_count);
    }

    return false;  // No valid card number detected
}
