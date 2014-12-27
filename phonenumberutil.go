package libphonenumber

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"code.google.com/p/goprotobuf/proto"

	"github.com/yext/glog"
)

const (
	// The minimum and maximum length of the national significant number.
	MIN_LENGTH_FOR_NSN = 2
	// The ITU says the maximum length should be 15, but we have
	// found longer numbers in Germany.
	MAX_LENGTH_FOR_NSN = 17
	// The maximum length of the country calling code.
	MAX_LENGTH_COUNTRY_CODE = 3
	// We don't allow input strings for parsing to be longer than 250 chars.
	// This prevents malicious input from overflowing the regular-expression
	// engine.
	MAX_INPUT_STRING_LENGTH = 250

	//META_DATA_FILE_PREFIX = "/com/google/i18n/phonenumbers/data/PhoneNumberMetadataProto"
	META_DATA_FILE_PREFIX = "/Users/ltacon/Documents/gocode/src/github.com/ttacon/libphonenumber/data/PhoneNumberMetadataProto"

	// Region-code for the unknown region.
	UNKNOWN_REGION = "ZZ"

	NANPA_COUNTRY_CODE = 1

	// The prefix that needs to be inserted in front of a Colombian
	// landline number when dialed from a mobile phone in Colombia.
	COLOMBIA_MOBILE_TO_FIXED_LINE_PREFIX = "3"

	// The PLUS_SIGN signifies the international prefix.
	PLUS_SIGN = '+'

	STAR_SIGN = '*'

	RFC3966_EXTN_PREFIX     = ";ext="
	RFC3966_PREFIX          = "tel:"
	RFC3966_PHONE_CONTEXT   = ";phone-context="
	RFC3966_ISDN_SUBADDRESS = ";isub="

	// Regular expression of acceptable punctuation found in phone
	// numbers. This excludes punctuation found as a leading character
	// only. This consists of dash characters, white space characters,
	// full stops, slashes, square brackets, parentheses and tildes. It
	// also includes the letter 'x' as that is found as a placeholder
	// for carrier information in some phone numbers. Full-width variants
	// are also present.
	VALID_PUNCTUATION = "-x\u2010-\u2015\u2212\u30FC\uFF0D-\uFF0F " +
		"\u00A0\u00AD\u200B\u2060\u3000()\uFF08\uFF09\uFF3B\uFF3D." +
		"\\[\\]/~\u2053\u223C\uFF5E"

	DIGITS = "\\p{Nd}"

	// We accept alpha characters in phone numbers, ASCII only, upper
	// and lower case.
	VALID_ALPHA = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	PLUS_CHARS  = "+\uFF0B"
)

var (
	defaultMetadataLoader MetadataLoader

	// Map of country calling codes that use a mobile token before the
	// area code. One example of when this is relevant is when determining
	// the length of the national destination code, which should be the
	// length of the area code plus the length of the mobile token.
	MOBILE_TOKEN_MAPPINGS = map[int]string{
		52: "1",
		54: "9",
	}

	// A map that contains characters that are essential when dialling.
	// That means any of the characters in this map must not be removed
	// from a number when dialling, otherwise the call will not reach
	// the intended destination.
	DIALLABLE_CHAR_MAPPINGS = map[rune]rune{
		'1':       '1',
		'2':       '2',
		'3':       '3',
		'4':       '4',
		'5':       '5',
		'6':       '6',
		'7':       '7',
		'8':       '8',
		'9':       '9',
		'0':       '0',
		PLUS_SIGN: PLUS_SIGN,
		'*':       '*',
	}

	// Only upper-case variants of alpha characters are stored.
	ALPHA_MAPPINGS = map[rune]rune{
		'A': '2',
		'B': '2',
		'C': '2',
		'D': '3',
		'E': '3',
		'F': '3',
		'G': '4',
		'H': '4',
		'I': '4',
		'J': '5',
		'K': '5',
		'L': '5',
		'M': '6',
		'N': '6',
		'O': '6',
		'P': '7',
		'Q': '7',
		'R': '7',
		'S': '7',
		'T': '8',
		'U': '8',
		'V': '8',
		'W': '9',
		'X': '9',
		'Y': '9',
		'Z': '9',
	}

	// For performance reasons, amalgamate both into one map.
	ALPHA_PHONE_MAPPINGS = map[rune]rune{
		'1':       '1',
		'2':       '2',
		'3':       '3',
		'4':       '4',
		'5':       '5',
		'6':       '6',
		'7':       '7',
		'8':       '8',
		'9':       '9',
		'0':       '0',
		PLUS_SIGN: PLUS_SIGN,
		'*':       '*',
		'A':       '2',
		'B':       '2',
		'C':       '2',
		'D':       '3',
		'E':       '3',
		'F':       '3',
		'G':       '4',
		'H':       '4',
		'I':       '4',
		'J':       '5',
		'K':       '5',
		'L':       '5',
		'M':       '6',
		'N':       '6',
		'O':       '6',
		'P':       '7',
		'Q':       '7',
		'R':       '7',
		'S':       '7',
		'T':       '8',
		'U':       '8',
		'V':       '8',
		'W':       '9',
		'X':       '9',
		'Y':       '9',
		'Z':       '9',
	}

	// Separate map of all symbols that we wish to retain when formatting
	// alpha numbers. This includes digits, ASCII letters and number
	// grouping symbols such as "-" and " ".
	ALL_PLUS_NUMBER_GROUPING_SYMBOLS = map[rune]rune{
		'1':       '1',
		'2':       '2',
		'3':       '3',
		'4':       '4',
		'5':       '5',
		'6':       '6',
		'7':       '7',
		'8':       '8',
		'9':       '9',
		'0':       '0',
		PLUS_SIGN: PLUS_SIGN,
		'*':       '*',
		'A':       'A',
		'B':       'B',
		'C':       'C',
		'D':       'D',
		'E':       'E',
		'F':       'F',
		'G':       'G',
		'H':       'H',
		'I':       'I',
		'J':       'J',
		'K':       'K',
		'L':       'L',
		'M':       'M',
		'N':       'N',
		'O':       'O',
		'P':       'P',
		'Q':       'Q',
		'R':       'R',
		'S':       'S',
		'T':       'T',
		'U':       'U',
		'V':       'V',
		'W':       'W',
		'X':       'X',
		'Y':       'Y',
		'Z':       'Z',
		'a':       'A',
		'b':       'B',
		'c':       'C',
		'd':       'D',
		'e':       'E',
		'f':       'F',
		'g':       'G',
		'h':       'H',
		'i':       'I',
		'j':       'J',
		'k':       'K',
		'l':       'L',
		'm':       'M',
		'n':       'N',
		'o':       'O',
		'p':       'P',
		'q':       'Q',
		'r':       'R',
		's':       'S',
		't':       'T',
		'u':       'U',
		'v':       'V',
		'w':       'W',
		'x':       'X',
		'y':       'Y',
		'z':       'Z',
		'-':       '-',
		'\uFF0D':  '-',
		'\u2010':  '-',
		'\u2011':  '-',
		'\u2012':  '-',
		'\u2013':  '-',
		'\u2014':  '-',
		'\u2015':  '-',
		'\u2212':  '-',
		'/':       '/',
		'\uFF0F':  '/',
		' ':       ' ',
		'\u3000':  ' ',
		'\u2060':  ' ',
		'.':       '.',
		'\uFF0E':  '.',
	}

	// Pattern that makes it easy to distinguish whether a region has a
	// unique international dialing prefix or not. If a region has a
	// unique international prefix (e.g. 011 in USA), it will be
	// represented as a string that contains a sequence of ASCII digits.
	// If there are multiple available international prefixes in a
	// region, they will be represented as a regex string that always
	// contains character(s) other than ASCII digits.
	// Note this regex also includes tilde, which signals waiting for the tone.
	UNIQUE_INTERNATIONAL_PREFIX = regexp.MustCompile("[\\d]+(?:[~\u2053\u223C\uFF5E][\\d]+)?")

	PLUS_CHARS_PATTERN      = regexp.MustCompile("[" + PLUS_CHARS + "]+")
	SEPARATOR_PATTERN       = regexp.MustCompile("[" + VALID_PUNCTUATION + "]+")
	CAPTURING_DIGIT_PATTERN = regexp.MustCompile("(" + DIGITS + ")")

	// Regular expression of acceptable characters that may start a
	// phone number for the purposes of parsing. This allows us to
	// strip away meaningless prefixes to phone numbers that may be
	// mistakenly given to us. This consists of digits, the plus symbol
	// and arabic-indic digits. This does not contain alpha characters,
	// although they may be used later in the number. It also does not
	// include other punctuation, as this will be stripped later during
	// parsing and is of no information value when parsing a number.
	VALID_START_CHAR         = "[" + PLUS_CHARS + DIGITS + "]"
	VALID_START_CHAR_PATTERN = regexp.MustCompile(VALID_START_CHAR)

	// Regular expression of characters typically used to start a second
	// phone number for the purposes of parsing. This allows us to strip
	// off parts of the number that are actually the start of another
	// number, such as for: (530) 583-6985 x302/x2303 -> the second
	// extension here makes this actually two phone numbers,
	// (530) 583-6985 x302 and (530) 583-6985 x2303. We remove the second
	// extension so that the first number is parsed correctly.
	SECOND_NUMBER_START         = "[\\\\/] *x"
	SECOND_NUMBER_START_PATTERN = regexp.MustCompile(SECOND_NUMBER_START)

	// Regular expression of trailing characters that we want to remove.
	// We remove all characters that are not alpha or numerical characters.
	// The hash character is retained here, as it may signify the previous
	// block was an extension.
	UNWANTED_END_CHARS        = "[[\\P{N}&&\\P{L}]&&[^#]]+$"
	UNWANTED_END_CHAR_PATTERN = regexp.MustCompile(UNWANTED_END_CHARS)

	// We use this pattern to check if the phone number has at least three
	// letters in it - if so, then we treat it as a number where some
	// phone-number digits are represented by letters.
	VALID_ALPHA_PHONE_PATTERN = regexp.MustCompile("(?:.*?[A-Za-z]){3}.*")

	// Regular expression of viable phone numbers. This is location
	// independent. Checks we have at least three leading digits, and
	// only valid punctuation, alpha characters and digits in the phone
	// number. Does not include extension data. The symbol 'x' is allowed
	// here as valid punctuation since it is often used as a placeholder
	// for carrier codes, for example in Brazilian phone numbers. We also
	// allow multiple "+" characters at the start.
	// Corresponds to the following:
	// [digits]{minLengthNsn}|
	// plus_sign*(
	//    ([punctuation]|[star])*[digits]
	// ){3,}([punctuation]|[star]|[digits]|[alpha])*
	//
	// The first reg-ex is to allow short numbers (two digits long) to be
	// parsed if they are entered as "15" etc, but only if there is no
	// punctuation in them. The second expression restricts the number of
	// digits to three or more, but then allows them to be in
	// international form, and to have alpha-characters and punctuation.
	//
	// Note VALID_PUNCTUATION starts with a -, so must be the first in the range.
	VALID_PHONE_NUMBER = DIGITS + "{" + strconv.Itoa(MIN_LENGTH_FOR_NSN) + "}" + "|" +
		"[" + PLUS_CHARS + "]*(?:[" + VALID_PUNCTUATION + string(STAR_SIGN) +
		"]*" + DIGITS + "){3,}[" +
		VALID_PUNCTUATION + string(STAR_SIGN) + VALID_ALPHA + DIGITS + "]*"

	// Default extension prefix to use when formatting. This will be put
	// in front of any extension component of the number, after the main
	// national number is formatted. For example, if you wish the default
	// extension formatting to be " extn: 3456", then you should specify
	// " extn: " here as the default extension prefix. This can be
	// overridden by region-specific preferences.
	DEFAULT_EXTN_PREFIX = " ext. "

	// Pattern to capture digits used in an extension. Places a maximum
	// length of "7" for an extension.
	CAPTURING_EXTN_DIGITS = "(" + DIGITS + "{1,7})"

	// Regexp of all possible ways to write extensions, for use when
	// parsing. This will be run as a case-insensitive regexp match.
	// Wide character versions are also provided after each ASCII version.
	// TODO(ttacon): okay so maybe some of these should go in an init?
	// There are three regular expressions here. The first covers RFC 3966
	// format, where the extension is added using ";ext=". The second more
	// generic one starts with optional white space and ends with an
	// optional full stop (.), followed by zero or more spaces/tabs and then
	// the numbers themselves. The other one covers the special case of
	// American numbers where the extension is written with a hash at the
	// end, such as "- 503#". Note that the only capturing groups should
	// be around the digits that you want to capture as part of the
	// extension, or else parsing will fail! Canonical-equivalence doesn't
	// seem to be an option with Android java, so we allow two options
	// for representing the accented o - the character itself, and one in
	// the unicode decomposed form with the combining acute accent.
	EXTN_PATTERNS_FOR_PARSING = RFC3966_EXTN_PREFIX + CAPTURING_EXTN_DIGITS + "|" + "[ \u00A0\\t,]*" +
		"(?:e?xt(?:ensi(?:o\u0301?|\u00F3))?n?|\uFF45?\uFF58\uFF54\uFF4E?|" +
		"[,x\uFF58#\uFF03~\uFF5E]|int|anexo|\uFF49\uFF4E\uFF54)" +
		"[:\\.\uFF0E]?[ \u00A0\\t,-]*" + CAPTURING_EXTN_DIGITS + "#?|" +
		"[- ]+(" + DIGITS + "{1,5})#"
	EXTN_PATTERNS_FOR_MATCHING = RFC3966_EXTN_PREFIX + CAPTURING_EXTN_DIGITS + "|" + "[ \u00A0\\t,]*" +
		"(?:e?xt(?:ensi(?:o\u0301?|\u00F3))?n?|\uFF45?\uFF58\uFF54\uFF4E?|" +
		"[x\uFF58#\uFF03~\uFF5E]|int|anexo|\uFF49\uFF4E\uFF54)" +
		"[:\\.\uFF0E]?[ \u00A0\\t,-]*" + CAPTURING_EXTN_DIGITS + "#?|" +
		"[- ]+(" + DIGITS + "{1,5})#"

	// Regexp of all known extension prefixes used by different regions
	// followed by 1 or more valid digits, for use when parsing.
	EXTN_PATTERN = regexp.MustCompile("(?:" + EXTN_PATTERNS_FOR_PARSING + ")$")

	// We append optionally the extension pattern to the end here, as a
	// valid phone number may have an extension prefix appended,
	// followed by 1 or more digits.
	VALID_PHONE_NUMBER_PATTERN = regexp.MustCompile(
		VALID_PHONE_NUMBER + "(?:" + EXTN_PATTERNS_FOR_PARSING + ")?")

	NON_DIGITS_PATTERN = regexp.MustCompile("(\\D+)")

	// The FIRST_GROUP_PATTERN was originally set to $1 but there are some
	// countries for which the first group is not used in the national
	// pattern (e.g. Argentina) so the $1 group does not match correctly.
	// Therefore, we use \d, so that the first group actually used in the
	// pattern will be matched.
	FIRST_GROUP_PATTERN = regexp.MustCompile("(\\$\\d)")
	NP_PATTERN          = regexp.MustCompile("\\$NP")
	FG_PATTERN          = regexp.MustCompile("\\$FG")
	CC_PATTERN          = regexp.MustCompile("\\$CC")

	// A pattern that is used to determine if the national prefix
	// formatting rule has the first group only, i.e., does not start
	// with the national prefix. Note that the pattern explicitly allows
	// for unbalanced parentheses.
	FIRST_GROUP_ONLY_PREFIX_PATTERN = regexp.MustCompile("\\(?\\$1\\)?")

	REGION_CODE_FOR_NON_GEO_ENTITY = "001"
)

// singleton pattern?
// private static PhoneNumberUtil instance = null;
// TODO(ttacon): make this go routine safe
var instance *PhoneNumberUtil

// INTERNATIONAL and NATIONAL formats are consistent with the definition
// in ITU-T Recommendation E123. For example, the number of the Google
// Switzerland office will be written as "+41 44 668 1800" in
// INTERNATIONAL format, and as "044 668 1800" in NATIONAL format. E164
// format is as per INTERNATIONAL format but with no formatting applied,
// e.g. "+41446681800". RFC3966 is as per INTERNATIONAL format, but with
// all spaces and other separating symbols replaced with a hyphen, and
// with any phone number extension appended with ";ext=". It also will
// have a prefix of "tel:" added, e.g. "tel:+41-44-668-1800".
//
// Note: If you are considering storing the number in a neutral format,
// you are highly advised to use the PhoneNumber class.

type PhoneNumberFormat int

const (
	E164 PhoneNumberFormat = iota
	INTERNATIONAL
	NATIONAL
	RFC3966
)

type PhoneNumberType int

const (
	// NOTES:
	//
	// FIXED_LINE_OR_MOBILE:
	//     In some regions (e.g. the USA), it is impossible to distinguish
	//     between fixed-line and mobile numbers by looking at the phone
	//     number itself.
	// SHARED_COST:
	//     The cost of this call is shared between the caller and the
	//     recipient, and is hence typically less than PREMIUM_RATE calls.
	//     See // http://en.wikipedia.org/wiki/Shared_Cost_Service for
	//     more information.
	// VOIP:
	//     Voice over IP numbers. This includes TSoIP (Telephony Service over IP).
	// PERSONAL_NUMBER:
	//     A personal number is associated with a particular person, and may
	//     be routed to either a MOBILE or FIXED_LINE number. Some more
	//     information can be found here:
	//     http://en.wikipedia.org/wiki/Personal_Numbers
	// UAN:
	//     Used for "Universal Access Numbers" or "Company Numbers". They
	//     may be further routed to specific offices, but allow one number
	//     to be used for a company.
	// VOICEMAIL:
	//     Used for "Voice Mail Access Numbers".
	// UNKNOWN:
	//     A phone number is of type UNKNOWN when it does not fit any of
	// the known patterns for a specific region.
	FIXED_LINE PhoneNumberType = iota
	MOBILE
	FIXED_LINE_OR_MOBILE
	TOLL_FREE
	PREMIUM_RATE
	SHARED_COST
	VOIP
	PERSONAL_NUMBER
	PAGER
	UAN
	VOICEMAIL
	UNKNOWN
)

type MatchType int

const (
	NOT_A_NUMBER MatchType = iota
	NO_MATCH
	SHORT_NSN_MATCH
	NSN_MATCH
	EXACT_MATCH
)

type ValidationResult int

const (
	IS_POSSIBLE ValidationResult = iota
	INVALID_COUNTRY_CODE
	TOO_SHORT
	TOO_LONG
)

// TODO(ttacon): leniency comments?
type Leniency int

const (
	POSSIBLE Leniency = iota
	VALID
	STRICT_GROUPING
	EXACT_GROUPING
)

func (l Leniency) Verify(
	number *PhoneNumber,
	candidate string,
	util *PhoneNumberUtil) bool {

	switch l {
	case POSSIBLE:
		return isPossibleNumber(number)
	case VALID:
		if !isValidNumber(number) ||
			!ContainsOnlyValidXChars(number, candidate, util) {
			return false
		}
		return IsNationalPrefixPresentIfRequired(number, util)
	case STRICT_GROUPING:
		if !isValidNumber(number) ||
			!ContainsOnlyValidXChars(number, candidate, util) ||
			ContainsMoreThanOneSlashInNationalNumber(number, candidate) ||
			!IsNationalPrefixPresentIfRequired(number, util) {
			return false
		}
		return CheckNumberGroupingIsValid(number, candidate, util,
			func(util *PhoneNumberUtil, number *PhoneNumber,
				normalizedCandidate string,
				expectedNumberGroups []string) bool {
				return AllNumberGroupsRemainGrouped(
					util, number, normalizedCandidate, expectedNumberGroups)
			})
	case EXACT_GROUPING:
		if !isValidNumber(number) ||
			!ContainsOnlyValidXChars(number, candidate, util) ||
			ContainsMoreThanOneSlashInNationalNumber(number, candidate) ||
			!IsNationalPrefixPresentIfRequired(number, util) {
			return false
		}
		return CheckNumberGroupingIsValid(number, candidate, util,
			func(util *PhoneNumberUtil, number *PhoneNumber,
				normalizedCandidate string,
				expectedNumberGroups []string) bool {
				return AllNumberGroupsAreExactlyPresent(
					util, number, normalizedCandidate, expectedNumberGroups)
			})
	}
	return false
}

var (
	// The set of regions that share country calling code 1.
	// There are roughly 26 regions.
	// We set the initial capacity of the HashSet to 35 to offer a load
	// factor of roughly 0.75.
	// TODO(ttacon): specify size?
	nanpaRegions = make(map[string]struct{})

	// A mapping from a region code to the PhoneMetadata for that region.
	// Note: Synchronization, though only needed for the Android version
	// of the library, is used in all versions for consistency.
	regionToMetadataMap = make(map[string]*PhoneMetadata)

	// A mapping from a country calling code for a non-geographical
	// entity to the PhoneMetadata for that country calling code.
	// Examples of the country calling codes include 800 (International
	// Toll Free Service) and 808 (International Shared Cost Service).
	// Note: Synchronization, though only needed for the Android version
	// of the library, is used in all versions for consistency.
	countryCodeToNonGeographicalMetadataMap = make(map[int]*PhoneMetadata)

	// A cache for frequently used region-specific regular expressions.
	// The initial capacity is set to 100 as this seems to be an optimal
	// value for Android, based on performance measurements.
	regexCache = make(map[string]*regexp.Regexp)

	// The set of regions the library supports.
	// There are roughly 240 of them and we set the initial capacity of
	// the HashSet to 320 to offer a load factor of roughly 0.75.
	supportedRegions = make(map[string]struct{})

	// The set of county calling codes that ma`p to the non-geo entity
	// region ("001"). This set currently contains < 12 elements so the
	// default capacity of 16 (load factor=0.75) is fine.
	countryCodesForNonGeographicalRegion = make(map[int]struct{})
)

type PhoneNumberUtil struct {
	// The prefix of the metadata files from which region data is loaded.
	currentFilePrefix string

	// The metadata loader used to inject alternative metadata sources.
	metadataLoader MetadataLoader

	// A mapping from a country calling code to the region codes which
	// denote the region represented by that country calling code. In
	// the case of multiple regions sharing a calling code, such as the
	// NANPA regions, the one indicated with "isMainCountryForCode" in
	// the metadata should be first.
	countryCallingCodeToRegionCodeMap map[int][]string
}

// This class implements a singleton, the constructor is only visible to
// facilitate testing.
// @VisibleForTesting
func NewPhoneNumberUtil(filePrefix string,
	metadataLoader MetadataLoader,
	countryCallingCodeToRegionCodeMap map[int][]string) *PhoneNumberUtil {

	this := &PhoneNumberUtil{
		currentFilePrefix:                 filePrefix,
		metadataLoader:                    metadataLoader,
		countryCallingCodeToRegionCodeMap: countryCallingCodeToRegionCodeMap,
	}

	for eKey, regionCodes := range countryCallingCodeToRegionCodeMap {
		// We can assume that if the county calling code maps to the
		// non-geo entity region code then that's the only region code
		// it maps to.
		if len(regionCodes) == 1 && REGION_CODE_FOR_NON_GEO_ENTITY == regionCodes[0] {
			// This is the subset of all country codes that map to the
			// non-geo entity region code.
			countryCodesForNonGeographicalRegion[eKey] = struct{}{}
		} else {
			// The supported regions set does not include the "001"
			// non-geo entity region code.
			for _, val := range regionCodes {
				supportedRegions[val] = struct{}{}
			}
		}
	}
	// If the non-geo entity still got added to the set of supported
	// regions it must be because there are entries that list the non-geo
	// entity alongside normal regions (which is wrong). If we discover
	// this, remove the non-geo entity from the set of supported regions
	// and log.
	if _, ok := supportedRegions[REGION_CODE_FOR_NON_GEO_ENTITY]; ok {
		glog.Warning("invalid metadata (country calling code was " +
			"mapped to the non-geo entity as well as specific region(s))")
	}
	for _, regions := range countryCallingCodeToRegionCodeMap {
		for _, val := range regions {
			nanpaRegions[val] = struct{}{}
		}
	}
	return this
}

// @VisibleForTesting
func loadMetadataFromFile(filePrefix, regionCode string,
	countryCallingCode int,
	metadataLoader MetadataLoader) error {

	isNonGeoRegion := REGION_CODE_FOR_NON_GEO_ENTITY == regionCode

	fileName := filePrefix + "_" + regionCode
	if isNonGeoRegion {
		fileName = filePrefix + "_" + strconv.Itoa(countryCallingCode)
	}

	metadataCollection, err := loadMetadataAndCloseInput(nil)
	if err != nil {
		return nil
	}

	metadataList := metadataCollection.GetMetadata()
	if len(metadataList) == 0 {
		glog.Error("empty metadata: " + fileName)
		return errors.New("empty metadata: " + fileName)
	}

	for _, meta := range metadataList {
		region := meta.GetId()
		regionToMetadataMap[region] = meta
	}

	// TODO(ttacon): nongeos?
	//	fmt.Println("isNonGeoRegion: ", isNonGeoRegion)
	//	if isNonGeoRegion {
	//		countryCodeToNonGeographicalMetadataMap[countryCallingCode] = metadata
	//	} else {
	//		regionToMetadataMap[regionCode] = metadata
	//	}
	return nil
}

// Loads the metadata protocol buffer from the given stream and closes
// the stream afterwards. Any exceptions that occur while reading the
// stream are propagated (though exceptions that occur when the stream
// is closed will be ignored).
func loadMetadataAndCloseInput(source io.ReadCloser) (*PhoneMetadataCollection, error) {
	metadataCollection := &PhoneMetadataCollection{}
	//data, err := ioutil.ReadAll(source)
	//if err != nil {
	//	return nil, err
	//}
	err := proto.Unmarshal(metaData, metadataCollection)
	return metadataCollection, err
}

// Attempts to extract a possible number from the string passed in.
// This currently strips all leading characters that cannot be used to
// start a phone number. Characters that can be used to start a phone
// number are defined in the VALID_START_CHAR_PATTERN. If none of these
// characters are found in the number passed in, an empty string is
// returned. This function also attempts to strip off any alternative
// extensions or endings if two or more are present, such as in the case
// of: (530) 583-6985 x302/x2303. The second extension here makes this
// actually two phone numbers, (530) 583-6985 x302 and (530) 583-6985 x2303.
// We remove the second extension so that the first number is parsed correctly.
func extractPossibleNumber(number string) string {
	if VALID_START_CHAR_PATTERN.MatchString(number) {
		// TODO(ttacon): is this right?
		start := VALID_START_CHAR_PATTERN.FindIndex([]byte(number))[0]
		number = number[start:]
		// Remove trailing non-alpha non-numerical characters.
		indices := UNWANTED_END_CHAR_PATTERN.FindIndex([]byte(number))
		if len(indices) > 0 {
			number = number[0:indices[0]]
			glog.Info("Stripped trailing characters: " + number)
		}
		// Check for extra numbers at the end.
		indices = SECOND_NUMBER_START_PATTERN.FindIndex([]byte(number))
		if len(indices) > 0 {
			number = number[0:indices[0]]
		}
		return number
	}
	return ""
}

// Checks to see if the string of characters could possibly be a phone
// number at all. At the moment, checks to see that the string begins
// with at least 2 digits, ignoring any punctuation commonly found in
// phone numbers. This method does not require the number to be
// normalized in advance - but does assume that leading non-number symbols
// have been removed, such as by the method extractPossibleNumber.
// @VisibleForTesting
func isViablePhoneNumber(number string) bool {
	if len(number) < MIN_LENGTH_FOR_NSN {
		return false
	}
	return VALID_PHONE_NUMBER_PATTERN.MatchString(number)
}

// Normalizes a string of characters representing a phone number. This
// performs the following conversions:
//   - Punctuation is stripped.
//   - For ALPHA/VANITY numbers:
//   - Letters are converted to their numeric representation on a telephone
//     keypad. The keypad used here is the one defined in ITU Recommendation
//     E.161. This is only done if there are 3 or more letters in the
//     number, to lessen the risk that such letters are typos.
//
// For other numbers:
//   - Wide-ascii digits are converted to normal ASCII (European) digits.
//   - Arabic-Indic numerals are converted to European numerals.
//   - Spurious alpha characters are stripped.
func normalize(number string) string {
	if VALID_ALPHA_PHONE_PATTERN.MatchString(number) {
		return normalizeHelper(number, ALPHA_PHONE_MAPPINGS, true)
	}
	return normalizeDigitsOnly(number)
}

// Normalizes a string of characters representing a phone number. This is
// a wrapper for normalize(String number) but does in-place normalization
// of the StringBuilder provided.
func normalizeBytes(number *bytes.Buffer) *bytes.Buffer {
	normalizedNumber := normalize(number.String())
	b := number.Bytes()
	copy(b[0:len(normalizedNumber)], []byte(normalizedNumber))
	return bytes.NewBuffer(b)
}

// Normalizes a string of characters representing a phone number. This
// converts wide-ascii and arabic-indic numerals to European numerals,
// and strips punctuation and alpha characters.
func normalizeDigitsOnly(number string) string {
	return string(normalizeDigits(number, false /* strip non-digits */))
}

// TODO(ttacon): add test for this versus java version...
// not sure it's implemented correctly
// URGENT(ttacon)
func normalizeDigits(number string, keepNonDigits bool) []byte {
	buf := []byte(number)
	var normalizedDigits []byte
	for _, c := range buf {
		// TODO(ttacon): make this not a dirty hack
		digit := c - 48
		if digit < 10 {
			normalizedDigits = append(normalizedDigits, c)
		} else if keepNonDigits {
			normalizedDigits = append(normalizedDigits, c)
		}
	}
	return normalizedDigits
}

// Normalizes a string of characters representing a phone number. This
// strips all characters which are not diallable on a mobile phone
// keypad (including all non-ASCII digits).
func normalizeDiallableCharsOnly(number string) string {
	return normalizeHelper(
		number, DIALLABLE_CHAR_MAPPINGS, true /* remove non matches */)
}

// Converts all alpha characters in a number to their respective digits
// on a keypad, but retains existing formatting.
func convertAlphaCharactersInNumber(number string) string {
	return normalizeHelper(number, ALPHA_PHONE_MAPPINGS, false)
}

// Gets the length of the geographical area code from the PhoneNumber
// object passed in, so that clients could use it to split a national
// significant number into geographical area code and subscriber number. It
// works in such a way that the resultant subscriber number should be
// diallable, at least on some devices. An example of how this could be used:
//
//   PhoneNumberUtil phoneUtil = PhoneNumberUtil.getInstance();
//   PhoneNumber number = phoneUtil.parse("16502530000", "US");
//   String nationalSignificantNumber = phoneUtil.getNationalSignificantNumber(number);
//   String areaCode;
//   String subscriberNumber;
//
//   int areaCodeLength = phoneUtil.getLengthOfGeographicalAreaCode(number);
//   if (areaCodeLength > 0) {
//     areaCode = nationalSignificantNumber.substring(0, areaCodeLength);
//     subscriberNumber = nationalSignificantNumber.substring(areaCodeLength);
//   } else {
//     areaCode = "";
//     subscriberNumber = nationalSignificantNumber;
//   }
//
// N.B.: area code is a very ambiguous concept, so the I18N team generally
// recommends against using it for most purposes, but recommends using the
// more general national_number instead. Read the following carefully before
// deciding to use this method:
//
//  - geographical area codes change over time, and this method honors those changes;
//    therefore, it doesn't guarantee the stability of the result it produces.
//  - subscriber numbers may not be diallable from all devices (notably mobile
//    devices, which typically requires the full national_number to be dialled
//    in most regions).
//  - most non-geographical numbers have no area codes, including numbers from
//    non-geographical entities
//  - some geographical numbers have no area codes.
func getLengthOfGeographicalAreaCode(number *PhoneNumber) int {
	metadata := getMetadataForRegion(getRegionCodeForNumber(number))
	if metadata == nil {
		return 0
	}

	// If a country doesn't use a national prefix, and this number
	// doesn't have an Italian leading zero, we assume it is a closed
	// dialling plan with no area codes.
	if len(metadata.GetNationalPrefix()) == 0 && !number.GetItalianLeadingZero() {
		return 0
	}

	if !isNumberGeographical(number) {
		return 0
	}

	return getLengthOfNationalDestinationCode(number)
}

// Gets the length of the national destination code (NDC) from the
// PhoneNumber object passed in, so that clients could use it to split a
// national significant number into NDC and subscriber number. The NDC of
// a phone number is normally the first group of digit(s) right after the
// country calling code when the number is formatted in the international
// format, if there is a subscriber number part that follows. An example
// of how this could be used:
//
//   PhoneNumberUtil phoneUtil = PhoneNumberUtil.getInstance();
//   PhoneNumber number = phoneUtil.parse("18002530000", "US");
//   String nationalSignificantNumber = phoneUtil.getNationalSignificantNumber(number);
//   String nationalDestinationCode;
//   String subscriberNumber;
//
//   int nationalDestinationCodeLength =
//       phoneUtil.getLengthOfNationalDestinationCode(number);
//   if nationalDestinationCodeLength > 0 {
//       nationalDestinationCode = nationalSignificantNumber.substring(0,
//           nationalDestinationCodeLength);
//       subscriberNumber = nationalSignificantNumber.substring(
//           nationalDestinationCodeLength);
//   } else {
//       nationalDestinationCode = "";
//       subscriberNumber = nationalSignificantNumber;
//   }
//
// Refer to the unittests to see the difference between this function and
// getLengthOfGeographicalAreaCode().
func getLengthOfNationalDestinationCode(number *PhoneNumber) int {
	var copiedProto *PhoneNumber
	if len(number.GetExtension()) > 0 {
		// We don't want to alter the proto given to us, but we don't
		// want to include the extension when we format it, so we copy
		// it and clear the extension here.
		copiedProto = &PhoneNumber{}
		proto.Merge(copiedProto, number)
		copiedProto.Extension = nil
	} else {
		copiedProto = number
	}

	nationalSignificantNumber := format(
		copiedProto, INTERNATIONAL)
	numberGroups := NON_DIGITS_PATTERN.FindAllString(nationalSignificantNumber, -1)
	// The pattern will start with "+COUNTRY_CODE " so the first group
	// will always be the empty string (before the + symbol) and the
	// second group will be the country calling code. The third group
	// will be area code if it is not the last group.
	if len(numberGroups) <= 3 {
		return 0
	}
	if getNumberType(number) == MOBILE {
		// For example Argentinian mobile numbers, when formatted in
		// the international format, are in the form of +54 9 NDC XXXX....
		// As a result, we take the length of the third group (NDC) and
		// add the length of the second group (which is the mobile token),
		// which also forms part of the national significant number. This
		// assumes that the mobile token is always formatted separately
		// from the rest of the phone number.
		mobileToken := getCountryMobileToken(int(number.GetCountryCode()))
		if mobileToken != "" {
			return len(numberGroups[2]) + len(numberGroups[3])
		}
	}
	return len(numberGroups[2])
}

// Returns the mobile token for the provided country calling code if it
// has one, otherwise returns an empty string. A mobile token is a number
// inserted before the area code when dialing a mobile number from that
// country from abroad.
func getCountryMobileToken(countryCallingCode int) string {
	if val, ok := MOBILE_TOKEN_MAPPINGS[countryCallingCode]; ok {
		return val
	}
	return ""
}

// Normalizes a string of characters representing a phone number by replacing
// all characters found in the accompanying map with the values therein,
// and stripping all other characters if removeNonMatches is true.
func normalizeHelper(number string,
	normalizationReplacements map[rune]rune,
	removeNonMatches bool) string {

	// TODO(ttacon): would using bytes.Buffer be faster? write a benchmark...
	var normalizedNumber = bytes.NewBuffer(nil)
	for _, character := range number {
		newDigit, ok := normalizationReplacements[unicode.ToUpper(character)]
		if ok {
			normalizedNumber.WriteRune(newDigit)
		} else if !removeNonMatches {
			normalizedNumber.WriteRune(character)
		}
		// If neither of the above are true, we remove this character.
	}
	return normalizedNumber.String()
}

// Sets or resets the PhoneNumberUtil singleton instance. If set to null,
// the next call to getInstance() will load (and return) the default instance.
// @VisibleForTesting
// TODO(ttacon): make a go routine safe singleton instance
func setInstance(util *PhoneNumberUtil) {
	instance = util
}

// Convenience method to get a list of what regions the library has metadata for.
// TODO(ttacon): make an unmodifiable wrapper (read-only)
func getSupportedRegions() map[string]struct{} {
	return supportedRegions
}

// Convenience method to get a list of what global network calling codes
// the library has metadata for.
// TODO(ttacon): make an unmodifiable wrapper (read-only)
func getSupportedGlobalNetworkCallingCodes() map[int]struct{} {
	return countryCodesForNonGeographicalRegion
}

// Gets a PhoneNumberUtil instance to carry out international phone number
// formatting, parsing, or validation. The instance is loaded with phone
// number metadata for a number of most commonly used regions.
//
// The PhoneNumberUtil is implemented as a singleton. Therefore, calling
// getInstance() multiple times will only result in one instance being created.
// TODO(ttacon): make this go routine safe
func getInstance() *PhoneNumberUtil {
	if instance == nil {
		setInstance(createInstance(DEFAULT_METADATA_LOADER))
	}
	return instance
}

// Create a new {@link PhoneNumberUtil} instance to carry out international
// phone number formatting, parsing, or validation. The instance is loaded
// with all metadata by using the metadataLoader specified.
//
// This method should only be used in the rare case in which you want to
// manage your own metadata loading. Calling this method multiple times is
// very expensive, as each time a new instance is created from scratch.
// When in doubt, use getInstance().
func createInstance(metadataLoader MetadataLoader) *PhoneNumberUtil {
	if metadataLoader == nil {
		panic("metadataLoader could not be null.")
	}
	return NewPhoneNumberUtil(
		META_DATA_FILE_PREFIX,
		metadataLoader,
		CountryCodeToRegion,
	)
}

// Helper function to check if the national prefix formatting rule has the
// first group only, i.e., does not start with the national prefix.
func formattingRuleHasFirstGroupOnly(nationalPrefixFormattingRule string) bool {
	return len(nationalPrefixFormattingRule) == 0 ||
		FIRST_GROUP_ONLY_PREFIX_PATTERN.MatchString(nationalPrefixFormattingRule)
}

// Tests whether a phone number has a geographical association. It checks
// if the number is associated to a certain region in the country where it
// belongs to. Note that this doesn't verify if the number is actually in use.
//
// A similar method is implemented as PhoneNumberOfflineGeocoder.canBeGeocoded,
// which performs a looser check, since it only prevents cases where prefixes
// overlap for geocodable and non-geocodable numbers. Also, if new phone
// number types were added, we should check if this other method should be
// updated too.
func isNumberGeographical(phoneNumber *PhoneNumber) bool {
	numberType := getNumberType(phoneNumber)
	// TODO: Include mobile phone numbers from countries like Indonesia,
	// which has some mobile numbers that are geographical.
	return numberType == FIXED_LINE ||
		numberType == FIXED_LINE_OR_MOBILE
}

// Helper function to check region code is not unknown or null.
func isValidRegionCode(regionCode string) bool {
	_, contains := supportedRegions[regionCode]
	return len(regionCode) != 0 && contains
}

// Helper function to check the country calling code is valid.
func hasValidCountryCallingCode(countryCallingCode int) bool {
	_, containsKey := CountryCodeToRegion[countryCallingCode]
	return containsKey
}

// Formats a phone number in the specified format using default rules. Note
// that this does not promise to produce a phone number that the user can
// dial from where they are - although we do format in either 'national' or
// 'international' format depending on what the client asks for, we do not
// currently support a more abbreviated format, such as for users in the
// same "area" who could potentially dial the number without area code.
// Note that if the phone number has a country calling code of 0 or an
// otherwise invalid country calling code, we cannot work out which
// formatting rules to apply so we return the national significant number
// with no formatting applied.
func format(number *PhoneNumber, numberFormat PhoneNumberFormat) string {
	if number.GetNationalNumber() == 0 && len(number.GetRawInput()) > 0 {
		// Unparseable numbers that kept their raw input just use that.
		// This is the only case where a number can be formatted as E164
		// without a leading '+' symbol (but the original number wasn't
		// parseable anyway).
		// TODO: Consider removing the 'if' above so that unparseable
		// strings without raw input format to the empty string instead of "+00"
		rawInput := number.GetRawInput()
		if len(rawInput) > 0 {
			return rawInput
		}
	}
	var formattedNumber = bytes.NewBuffer(nil)
	formatWithBuf(number, numberFormat, formattedNumber)
	return formattedNumber.String()
}

// Same as format(PhoneNumber, PhoneNumberFormat), but accepts a mutable
// StringBuilder as a parameter to decrease object creation when invoked
// many times.
func formatWithBuf(number *PhoneNumber, numberFormat PhoneNumberFormat,
	formattedNumber *bytes.Buffer) {
	// Clear the StringBuilder first.
	formattedNumber.Reset()
	countryCallingCode := int(number.GetCountryCode())
	nationalSignificantNumber := getNationalSignificantNumber(number)

	if numberFormat == E164 {
		// Early exit for E164 case (even if the country calling code
		// is invalid) since no formatting of the national number needs
		// to be applied. Extensions are not formatted.
		// TODO(ttacon): run through errcheck or something similar
		formattedNumber.WriteString(nationalSignificantNumber)
		prefixNumberWithCountryCallingCode(
			countryCallingCode,
			E164,
			formattedNumber)
		return
	}
	if !hasValidCountryCallingCode(countryCallingCode) {
		// TODO(ttacon): run through errcheck or something similar
		formattedNumber.WriteString(nationalSignificantNumber)
		return
	}
	// Note getRegionCodeForCountryCode() is used because formatting
	// information for regions which share a country calling code is
	// contained by only one region for performance reasons. For
	// example, for NANPA regions it will be contained in the metadata for US.
	regionCode := getRegionCodeForCountryCode(countryCallingCode)
	// Metadata cannot be null because the country calling code is
	// valid (which means that the region code cannot be ZZ and must
	// be one of our supported region codes).
	metadata := getMetadataForRegionOrCallingCode(
		countryCallingCode, regionCode)
	formattedNumber.WriteString(
		formatNsn(nationalSignificantNumber, metadata, numberFormat))
	maybeAppendFormattedExtension(number, metadata, numberFormat, formattedNumber)
	prefixNumberWithCountryCallingCode(
		countryCallingCode, numberFormat, formattedNumber)
}

// Formats a phone number in the specified format using client-defined
// formatting rules. Note that if the phone number has a country calling
// code of zero or an otherwise invalid country calling code, we cannot
// work out things like whether there should be a national prefix applied,
// or how to format extensions, so we return the national significant
// number with no formatting applied.
func formatByPattern(number *PhoneNumber,
	numberFormat PhoneNumberFormat,
	userDefinedFormats []*NumberFormat) string {

	countryCallingCode := int(number.GetCountryCode())
	nationalSignificantNumber := getNationalSignificantNumber(number)
	if !hasValidCountryCallingCode(countryCallingCode) {
		return nationalSignificantNumber
	}
	// Note getRegionCodeForCountryCode() is used because formatting
	// information for regions which share a country calling code is
	// contained by only one region for performance reasons. For example,
	// for NANPA regions it will be contained in the metadata for US.
	regionCode := getRegionCodeForCountryCode(countryCallingCode)
	// Metadata cannot be null because the country calling code is valid
	metadata := getMetadataForRegionOrCallingCode(countryCallingCode, regionCode)

	formattedNumber := bytes.NewBuffer(nil)

	formattingPattern := chooseFormattingPatternForNumber(
		userDefinedFormats, nationalSignificantNumber)
	if formattingPattern == nil {
		// If no pattern above is matched, we format the number as a whole.
		formattedNumber.WriteString(nationalSignificantNumber)
	} else {
		var numFormatCopy *NumberFormat
		// Before we do a replacement of the national prefix pattern
		// $NP with the national prefix, we need to copy the rule so
		// that subsequent replacements for different numbers have the
		// appropriate national prefix.
		proto.Merge(numFormatCopy, formattingPattern)
		nationalPrefixFormattingRule := formattingPattern.GetNationalPrefixFormattingRule()
		if len(nationalPrefixFormattingRule) > 0 {
			nationalPrefix := metadata.GetNationalPrefix()
			if len(nationalPrefix) > 0 {
				// Replace $NP with national prefix and $FG with the
				// first group ($1).
				nationalPrefixFormattingRule =
					NP_PATTERN.ReplaceAllString(
						nationalPrefixFormattingRule, nationalPrefix)
				nationalPrefixFormattingRule =
					FG_PATTERN.ReplaceAllString(
						nationalPrefixFormattingRule, "\\$1")
				numFormatCopy.NationalPrefixFormattingRule =
					&nationalPrefixFormattingRule
			} else {
				// We don't want to have a rule for how to format the
				// national prefix if there isn't one.
				numFormatCopy.NationalPrefixFormattingRule = nil
			}
		}
		formattedNumber.WriteString(
			formatNsnUsingPattern(
				nationalSignificantNumber, numFormatCopy, numberFormat))
	}
	maybeAppendFormattedExtension(number, metadata, numberFormat, formattedNumber)
	prefixNumberWithCountryCallingCode(countryCallingCode, numberFormat, formattedNumber)
	return formattedNumber.String()
}

// Formats a phone number in national format for dialing using the carrier
// as specified in the carrierCode. The carrierCode will always be used
// regardless of whether the phone number already has a preferred domestic
// carrier code stored. If carrierCode contains an empty string, returns
// the number in national format without any carrier code.
func formatNationalNumberWithCarrierCode(number *PhoneNumber, carrierCode string) string {
	countryCallingCode := int(number.GetCountryCode())
	nationalSignificantNumber := getNationalSignificantNumber(number)
	if !hasValidCountryCallingCode(countryCallingCode) {
		return nationalSignificantNumber
	}
	// Note getRegionCodeForCountryCode() is used because formatting
	// information for regions which share a country calling code is
	// contained by only one region for performance reasons. For
	// example, for NANPA regions it will be contained in the metadata for US.
	regionCode := getRegionCodeForCountryCode(countryCallingCode)
	// Metadata cannot be null because the country calling code is valid.
	metadata := getMetadataForRegionOrCallingCode(countryCallingCode, regionCode)

	formattedNumber := bytes.NewBuffer(nil)
	formattedNumber.WriteString(
		formatNsnWithCarrier(
			nationalSignificantNumber,
			metadata,
			NATIONAL,
			carrierCode))
	maybeAppendFormattedExtension(number, metadata, NATIONAL, formattedNumber)
	prefixNumberWithCountryCallingCode(
		countryCallingCode,
		NATIONAL,
		formattedNumber)
	return formattedNumber.String()
}

func getMetadataForRegionOrCallingCode(
	countryCallingCode int, regionCode string) *PhoneMetadata {
	if REGION_CODE_FOR_NON_GEO_ENTITY == regionCode {
		return getMetadataForNonGeographicalRegion(countryCallingCode)
	}
	return getMetadataForRegion(regionCode)
}

// Formats a phone number in national format for dialing using the carrier
// as specified in the preferredDomesticCarrierCode field of the PhoneNumber
// object passed in. If that is missing, use the fallbackCarrierCode passed
// in instead. If there is no preferredDomesticCarrierCode, and the
// fallbackCarrierCode contains an empty string, return the number in
// national format without any carrier code.
//
// Use formatNationalNumberWithCarrierCode instead if the carrier code
// passed in should take precedence over the number's
// preferredDomesticCarrierCode} when formatting.
func formatNationalNumberWithPreferredCarrierCode(
	number *PhoneNumber,
	fallbackCarrierCode string) string {

	pref := number.GetPreferredDomesticCarrierCode()
	if number.GetPreferredDomesticCarrierCode() == "" {
		pref = fallbackCarrierCode
	}
	return formatNationalNumberWithCarrierCode(number, pref)
}

// Returns a number formatted in such a way that it can be dialed from a
// mobile phone in a specific region. If the number cannot be reached from
// the region (e.g. some countries block toll-free numbers from being
// called outside of the country), the method returns an empty string.
func formatNumberForMobileDialing(
	number *PhoneNumber,
	regionCallingFrom string,
	withFormatting bool) string {

	countryCallingCode := int(number.GetCountryCode())
	if !hasValidCountryCallingCode(countryCallingCode) {
		return number.GetRawInput() // go impl defaults to ""
	}

	formattedNumber := ""
	// Clear the extension, as that part cannot normally be dialed
	// together with the main number.
	var numberNoExt *PhoneNumber
	proto.Merge(numberNoExt, number)
	numberNoExt.Extension = nil // can we assume this is safe? (no nil-pointer?)
	regionCode := getRegionCodeForCountryCode(countryCallingCode)
	numberType := getNumberType(numberNoExt)
	isValidNumber := numberType != UNKNOWN
	if regionCallingFrom == regionCode {
		isFixedLineOrMobile :=
			numberType == FIXED_LINE ||
				numberType == MOBILE ||
				numberType == FIXED_LINE_OR_MOBILE
		// Carrier codes may be needed in some countries. We handle this here.
		if regionCode == "CO" && numberType == FIXED_LINE {
			formattedNumber =
				formatNationalNumberWithCarrierCode(
					numberNoExt, COLOMBIA_MOBILE_TO_FIXED_LINE_PREFIX)
		} else if regionCode == "BR" && isFixedLineOrMobile {
			if numberNoExt.GetPreferredDomesticCarrierCode() != "" {
				formattedNumber =
					formatNationalNumberWithPreferredCarrierCode(numberNoExt, "")
			} else {
				// Brazilian fixed line and mobile numbers need to be dialed
				// with a carrier code when called within Brazil. Without
				// that, most of the carriers won't connect the call.
				// Because of that, we return an empty string here.
				formattedNumber = ""
			}
		} else if isValidNumber && regionCode == "HU" {
			// The national format for HU numbers doesn't contain the
			// national prefix, because that is how numbers are normally
			// written down. However, the national prefix is obligatory when
			// dialing from a mobile phone, except for short numbers. As a
			// result, we add it back here
			// if it is a valid regular length phone number.
			formattedNumber =
				getNddPrefixForRegion(regionCode, true /* strip non-digits */) +
					" " + format(numberNoExt, NATIONAL)
		} else if countryCallingCode == NANPA_COUNTRY_CODE {
			// For NANPA countries, we output international format for
			// numbers that can be dialed internationally, since that
			// always works, except for numbers which might potentially be
			// short numbers, which are always dialled in national format.
			regionMetadata := getMetadataForRegion(regionCallingFrom)
			if canBeInternationallyDialled(numberNoExt) &&
				!isShorterThanPossibleNormalNumber(regionMetadata,
					getNationalSignificantNumber(numberNoExt)) {
				formattedNumber = format(numberNoExt, INTERNATIONAL)
			} else {
				formattedNumber = format(numberNoExt, NATIONAL)
			}
		} else {
			// For non-geographical countries, and Mexican and Chilean fixed
			// line and mobile numbers, we output international format for
			// numbers that can be dialed internationally as that always
			// works.

			// MX fixed line and mobile numbers should always be formatted
			// in international format, even when dialed within MX. For
			// national format to work, a carrier code needs to be used,
			// and the correct carrier code depends on if the caller and
			// callee are from the same local area. It is trickier to get
			// that to work correctly than using international format, which
			// is tested to work fine on all carriers. CL fixed line
			// numbers need the national prefix when dialing in the national
			// format, but don't have it when used for display. The reverse
			// is true for mobile numbers. As a result, we output them in
			// the international format to make it work.
			if regionCode == REGION_CODE_FOR_NON_GEO_ENTITY ||
				((regionCode == "MX" ||
					regionCode == "CL") &&
					isFixedLineOrMobile) &&
					canBeInternationallyDialled(numberNoExt) {
				formattedNumber = format(numberNoExt, INTERNATIONAL)
			} else {
				formattedNumber = format(numberNoExt, NATIONAL)
			}
		}
	} else if isValidNumber && canBeInternationallyDialled(numberNoExt) {
		// We assume that short numbers are not diallable from outside
		// their region, so if a number is not a valid regular length
		// phone number, we treat it as if it cannot be internationally
		// dialled.
		if withFormatting {
			return format(numberNoExt, INTERNATIONAL)
		}
		return format(numberNoExt, E164)
	}
	if withFormatting {
		return formattedNumber
	}
	return normalizeDiallableCharsOnly(formattedNumber)
}

// Formats a phone number for out-of-country dialing purposes. If no
// regionCallingFrom is supplied, we format the number in its
// INTERNATIONAL format. If the country calling code is the same as that
// of the region where the number is from, then NATIONAL formatting will
// be applied.
//
// If the number itself has a country calling code of zero or an otherwise
// invalid country calling code, then we return the number with no
// formatting applied.
//
// Note this function takes care of the case for calling inside of NANPA and
// between Russia and Kazakhstan (who share the same country calling code).
// In those cases, no international prefix is used. For regions which have
// multiple international prefixes, the number in its INTERNATIONAL format
// will be returned instead.
func formatOutOfCountryCallingNumber(
	number *PhoneNumber,
	regionCallingFrom string) string {

	if !isValidRegionCode(regionCallingFrom) {
		glog.Warning(
			"Trying to format number from invalid region " +
				regionCallingFrom +
				". International formatting applied.")
		return format(number, INTERNATIONAL)
	}
	countryCallingCode := int(number.GetCountryCode())
	nationalSignificantNumber := getNationalSignificantNumber(number)
	if !hasValidCountryCallingCode(countryCallingCode) {
		return nationalSignificantNumber
	}
	if countryCallingCode == NANPA_COUNTRY_CODE {
		if isNANPACountry(regionCallingFrom) {
			// For NANPA regions, return the national format for these
			// regions but prefix it with the country calling code.
			return strconv.Itoa(countryCallingCode) + " " + format(number, NATIONAL)
		}
	} else if countryCallingCode == GetCountryCodeForValidRegion(regionCallingFrom) {
		// If regions share a country calling code, the country calling
		// code need not be dialled. This also applies when dialling
		// within a region, so this if clause covers both these cases.
		// Technically this is the case for dialling from La Reunion to
		// other overseas departments of France (French Guiana, Martinique,
		// Guadeloupe), but not vice versa - so we don't cover this edge
		// case for now and for those cases return the version including
		// country calling code.
		// Details here: http://www.petitfute.com/voyage/225-info-pratiques-reunion
		return format(number, NATIONAL)
	}
	// Metadata cannot be null because we checked 'isValidRegionCode()' above.
	metadataForRegionCallingFrom := getMetadataForRegion(regionCallingFrom)
	internationalPrefix := metadataForRegionCallingFrom.GetInternationalPrefix()

	// For regions that have multiple international prefixes, the
	// international format of the number is returned, unless there is
	// a preferred international prefix.
	internationalPrefixForFormatting := ""
	metPref := metadataForRegionCallingFrom.GetPreferredInternationalPrefix()
	if UNIQUE_INTERNATIONAL_PREFIX.MatchString(internationalPrefix) {
		internationalPrefixForFormatting = internationalPrefix
	} else if metPref != "" {
		internationalPrefixForFormatting = metPref
	}

	regionCode := getRegionCodeForCountryCode(countryCallingCode)
	// Metadata cannot be null because the country calling code is valid.
	metadataForRegion :=
		getMetadataForRegionOrCallingCode(countryCallingCode, regionCode)
	formattedNationalNumber :=
		formatNsn(
			nationalSignificantNumber, metadataForRegion, INTERNATIONAL)
	formattedNumber := bytes.NewBuffer([]byte(formattedNationalNumber))
	maybeAppendFormattedExtension(number, metadataForRegion, INTERNATIONAL,
		formattedNumber)
	if len(internationalPrefixForFormatting) > 0 {
		formattedBytes := formattedNumber.Bytes()
		formattedBytes = append([]byte(" "), formattedBytes...)
		// we know countryCallingCode is really an int32
		intBuf := []byte{
			byte(countryCallingCode >> 24),
			byte(countryCallingCode >> 16),
			byte(countryCallingCode >> 8),
			byte(countryCallingCode),
		}
		formattedBytes = append(intBuf, formattedBytes...)
		formattedBytes = append([]byte(" "), formattedBytes...)
		formattedBytes = append(
			[]byte(internationalPrefixForFormatting), formattedBytes...)
		return string(formattedBytes)
	} else {
		prefixNumberWithCountryCallingCode(
			countryCallingCode, INTERNATIONAL, formattedNumber)
	}
	return formattedNumber.String()
}

// Formats a phone number using the original phone number format that the
// number is parsed from. The original format is embedded in the
// country_code_source field of the PhoneNumber object passed in. If such
// information is missing, the number will be formatted into the NATIONAL
// format by default. When the number contains a leading zero and this is
// unexpected for this country, or we don't have a formatting pattern for
// the number, the method returns the raw input when it is available.
//
// Note this method guarantees no digit will be inserted, removed or
// modified as a result of formatting.
func formatInOriginalFormat(number *PhoneNumber, regionCallingFrom string) string {
	rawInput := number.GetRawInput()
	if len(rawInput) == 0 &&
		(hasUnexpectedItalianLeadingZero(number) ||
			!hasFormattingPatternForNumber(number)) {
		// We check if we have the formatting pattern because without that, we might format the number
		// as a group without national prefix.
		return rawInput
	}
	if number.GetCountryCodeSource() == 0 {
		return format(number, NATIONAL)
	}
	var formattedNumber string
	switch number.GetCountryCodeSource() {
	case PhoneNumber_FROM_NUMBER_WITH_PLUS_SIGN:
		formattedNumber = format(number, INTERNATIONAL)
	case PhoneNumber_FROM_NUMBER_WITH_IDD:
		formattedNumber = formatOutOfCountryCallingNumber(number, regionCallingFrom)
	case PhoneNumber_FROM_NUMBER_WITHOUT_PLUS_SIGN:
		formattedNumber = format(number, INTERNATIONAL)[1:]
	case PhoneNumber_FROM_DEFAULT_COUNTRY:
		// Fall-through to default case.
		fallthrough
	default:
		regionCode := getRegionCodeForCountryCode(int(number.GetCountryCode()))
		// We strip non-digits from the NDD here, and from the raw
		// input later, so that we can compare them easily.
		nationalPrefix := getNddPrefixForRegion(
			regionCode, true /* strip non-digits */)
		nationalFormat := format(number, NATIONAL)
		if len(nationalPrefix) == 0 || len(nationalPrefix) == 0 {
			// If the region doesn't have a national prefix at all,
			// we can safely return the national format without worrying
			// about a national prefix being added.
			formattedNumber = nationalFormat
			break
		}
		// Otherwise, we check if the original number was entered with
		// a national prefix.
		if rawInputContainsNationalPrefix(rawInput, nationalPrefix, regionCode) {
			// If so, we can safely return the national format.
			formattedNumber = nationalFormat
		}
		// Metadata cannot be null here because getNddPrefixForRegion()
		// (above) returns null if there is no metadata for the region.
		metadata := getMetadataForRegion(regionCode)
		nationalNumber := getNationalSignificantNumber(number)
		formatRule :=
			chooseFormattingPatternForNumber(metadata.GetNumberFormat(), nationalNumber)
		// The format rule could still be null here if the national
		// number was 0 and there was no raw input (this should not
		// be possible for numbers generated by the phonenumber library
		// as they would also not have a country calling code and we
		// would have exited earlier).
		if formatRule == nil {
			formattedNumber = nationalFormat
			break
		}
		// When the format we apply to this number doesn't contain
		// national prefix, we can just return the national format.
		// TODO: Refactor the code below with the code in
		// isNationalPrefixPresentIfRequired.
		candidateNationalPrefixRule := formatRule.GetNationalPrefixFormattingRule()
		// We assume that the first-group symbol will never be _before_
		// the national prefix.
		indexOfFirstGroup := strings.Index(candidateNationalPrefixRule, "$1")
		if indexOfFirstGroup <= 0 {
			formattedNumber = nationalFormat
			break
		}
		candidateNationalPrefixRule =
			candidateNationalPrefixRule[0:indexOfFirstGroup]
		candidateNationalPrefixRule = normalizeDigitsOnly(candidateNationalPrefixRule)
		if len(candidateNationalPrefixRule) == 0 {
			// National prefix not used when formatting this number.
			formattedNumber = nationalFormat
			break
		}
		// Otherwise, we need to remove the national prefix from our output.
		var numFormatCopy *NumberFormat
		proto.Merge(numFormatCopy, formatRule)
		numFormatCopy.NationalPrefixFormattingRule = nil
		var numberFormats = []*NumberFormat{numFormatCopy}
		formattedNumber = formatByPattern(number, NATIONAL, numberFormats)
		break
	}
	rawInput = number.GetRawInput()
	// If no digit is inserted/removed/modified as a result of our
	// formatting, we return the formatted phone number; otherwise we
	// return the raw input the user entered.
	if len(formattedNumber) != 0 && len(rawInput) > 0 {
		normalizedFormattedNumber := normalizeDiallableCharsOnly(formattedNumber)
		normalizedRawInput := normalizeDiallableCharsOnly(rawInput)
		if normalizedFormattedNumber != normalizedRawInput {
			formattedNumber = rawInput
		}
	}
	return formattedNumber
}

// Check if rawInput, which is assumed to be in the national format, has
// a national prefix. The national prefix is assumed to be in digits-only
// form.
func rawInputContainsNationalPrefix(rawInput, nationalPrefix, regionCode string) bool {
	normalizedNationalNumber := normalizeDigitsOnly(rawInput)
	if strings.HasPrefix(normalizedNationalNumber, nationalPrefix) {
		// Some Japanese numbers (e.g. 00777123) might be mistaken to
		// contain the national prefix when written without it
		// (e.g. 0777123) if we just do prefix matching. To tackle that,
		// we check the validity of the number if the assumed national
		// prefix is removed (777123 won't be valid in Japan).
		num, err := parse(normalizedNationalNumber[len(nationalPrefix):], regionCode)
		if err != nil {
			return false
		}
		return isValidNumber(num)

	}
	return false
}

// Returns true if a number is from a region whose national significant
// number couldn't contain a leading zero, but has the italian_leading_zero
// field set to true.
func hasUnexpectedItalianLeadingZero(number *PhoneNumber) bool {
	return number.GetItalianLeadingZero() &&
		!isLeadingZeroPossible(int(number.GetCountryCode()))
}

func hasFormattingPatternForNumber(number *PhoneNumber) bool {
	countryCallingCode := int(number.GetCountryCode())
	phoneNumberRegion := getRegionCodeForCountryCode(countryCallingCode)
	metadata := getMetadataForRegionOrCallingCode(
		countryCallingCode, phoneNumberRegion)
	if metadata == nil {
		return false
	}
	nationalNumber := getNationalSignificantNumber(number)
	formatRule := chooseFormattingPatternForNumber(
		metadata.GetNumberFormat(), nationalNumber)
	return formatRule != nil
}

// Formats a phone number for out-of-country dialing purposes.
//
// Note that in this version, if the number was entered originally using
// alpha characters and this version of the number is stored in raw_input,
// this representation of the number will be used rather than the digit
// representation. Grouping information, as specified by characters
// such as "-" and " ", will be retained.
//
// Caveats:
//
//  - This will not produce good results if the country calling code is
//    both present in the raw input _and_ is the start of the national
//    number. This is not a problem in the regions which typically use
//    alpha numbers.
//  - This will also not produce good results if the raw input has any
//    grouping information within the first three digits of the national
//    number, and if the function needs to strip preceding digits/words
//    in the raw input before these digits. Normally people group the
//    first three digits together so this is not a huge problem - and will
//    be fixed if it proves to be so.
func formatOutOfCountryKeepingAlphaChars(
	number *PhoneNumber,
	regionCallingFrom string) string {

	rawInput := number.GetRawInput()
	// If there is no raw input, then we can't keep alpha characters
	// because there aren't any. In this case, we return
	// formatOutOfCountryCallingNumber.
	if len(rawInput) == 0 {
		return formatOutOfCountryCallingNumber(number, regionCallingFrom)
	}
	countryCode := int(number.GetCountryCode())
	if !hasValidCountryCallingCode(countryCode) {
		return rawInput
	}
	// Strip any prefix such as country calling code, IDD, that was
	// present. We do this by comparing the number in raw_input with
	// the parsed number. To do this, first we normalize punctuation.
	// We retain number grouping symbols such as " " only.
	rawInput = normalizeHelper(rawInput, ALL_PLUS_NUMBER_GROUPING_SYMBOLS, true)
	// Now we trim everything before the first three digits in the
	// parsed number. We choose three because all valid alpha numbers
	// have 3 digits at the start - if it does not, then we don't trim
	// anything at all. Similarly, if the national number was less than
	// three digits, we don't trim anything at all.
	nationalNumber := getNationalSignificantNumber(number)
	if len(nationalNumber) > 3 {
		firstNationalNumberDigit := strings.Index(rawInput, nationalNumber[0:3])
		if firstNationalNumberDigit > -1 {
			rawInput = rawInput[firstNationalNumberDigit:]
		}
	}
	metadataForRegionCallingFrom := getMetadataForRegion(regionCallingFrom)
	if countryCode == NANPA_COUNTRY_CODE {
		if isNANPACountry(regionCallingFrom) {
			return strconv.Itoa(countryCode) + " " + rawInput
		}
	} else if metadataForRegionCallingFrom != nil &&
		countryCode == GetCountryCodeForValidRegion(regionCallingFrom) {
		formattingPattern :=
			chooseFormattingPatternForNumber(
				metadataForRegionCallingFrom.GetNumberFormat(),
				nationalNumber)
		if formattingPattern == nil {
			// If no pattern above is matched, we format the original input.
			return rawInput
		}
		var newFormat *NumberFormat
		proto.Merge(newFormat, formattingPattern)
		// The first group is the first group of digits that the user
		// wrote together.
		newFormat.Pattern = proto.String("(\\d+)(.*)")
		// Here we just concatenate them back together after the national
		// prefix has been fixed.
		newFormat.Format = proto.String("$1$2")
		// Now we format using this pattern instead of the default pattern,
		// but with the national prefix prefixed if necessary. This will not
		// work in the cases where the pattern (and not the leading digits)
		// decide whether a national prefix needs to be used, since we
		// have overridden the pattern to match anything, but that is not
		// the case in the metadata to date.
		return formatNsnUsingPattern(rawInput, newFormat, NATIONAL)
	}
	var internationalPrefixForFormatting = ""
	// If an unsupported region-calling-from is entered, or a country
	// with multiple international prefixes, the international format
	// of the number is returned, unless there is a preferred international
	// prefix.
	if metadataForRegionCallingFrom != nil {
		internationalPrefix := metadataForRegionCallingFrom.GetInternationalPrefix()
		internationalPrefixForFormatting = internationalPrefix
		if !UNIQUE_INTERNATIONAL_PREFIX.MatchString(internationalPrefix) {
			internationalPrefixForFormatting =
				metadataForRegionCallingFrom.GetPreferredInternationalPrefix()
		}
	}
	var formattedNumber = bytes.NewBuffer([]byte(rawInput))
	regionCode := getRegionCodeForCountryCode(countryCode)
	// Metadata cannot be null because the country calling code is valid.
	var metadataForRegion *PhoneMetadata = getMetadataForRegionOrCallingCode(countryCode, regionCode)
	maybeAppendFormattedExtension(number, metadataForRegion,
		INTERNATIONAL, formattedNumber)
	if len(internationalPrefixForFormatting) > 0 {
		formattedBytes := append([]byte(" "), formattedNumber.Bytes()...)
		// we know countryCode is really an int32
		intBuf := []byte{
			byte(countryCode >> 24),
			byte(countryCode >> 16),
			byte(countryCode >> 8),
			byte(countryCode),
		}
		formattedBytes = append(intBuf, formattedBytes...)
		formattedBytes = append([]byte(" "), formattedBytes...)
		formattedBytes = append(
			[]byte(internationalPrefixForFormatting), formattedBytes...)

		formattedNumber = bytes.NewBuffer(formattedBytes)
	} else {
		// Invalid region entered as country-calling-from (so no metadata
		// was found for it) or the region chosen has multiple international
		// dialling prefixes.
		glog.Warning(
			"Trying to format number from invalid region " +
				regionCallingFrom +
				". International formatting applied.")
		prefixNumberWithCountryCallingCode(countryCode,
			INTERNATIONAL,
			formattedNumber)
	}
	return formattedNumber.String()
}

// Gets the national significant number of the a phone number. Note a
// national significant number doesn't contain a national prefix or
// any formatting.
func getNationalSignificantNumber(number *PhoneNumber) string {
	// If leading zero(s) have been set, we prefix this now. Note this
	// is not a national prefix.
	nationalNumber := bytes.NewBuffer(nil)
	if number.GetItalianLeadingZero() {
		zeros := make([]byte, number.GetNumberOfLeadingZeros())
		for i := range zeros {
			zeros[i] = '0'
		}
		nationalNumber.Write(zeros)
	}
	natNum := number.GetNationalNumber()
	nationalNumber.Write([]byte{
		byte(natNum >> 56),
		byte(natNum >> 48),
		byte(natNum >> 40),
		byte(natNum >> 32),
		byte(natNum >> 24),
		byte(natNum >> 16),
		byte(natNum >> 8),
		byte(natNum & 0xff),
	})
	return nationalNumber.String()
}

// A helper function that is used by format and formatByPattern.
func prefixNumberWithCountryCallingCode(
	countryCallingCode int,
	numberFormat PhoneNumberFormat,
	formattedNumber *bytes.Buffer) {

	// TODO(ttacon): a lot of this will be super inefficient, build a package
	// that has a WriteAt()
	newBuf := bytes.NewBuffer(nil)
	switch numberFormat {
	case E164:
		newBuf.WriteString(string(PLUS_SIGN))
		newBuf.Write(strconv.AppendInt([]byte{}, int64(countryCallingCode), 10))
		newBuf.Write(formattedNumber.Bytes())
	case INTERNATIONAL:
		newBuf.WriteString(string(PLUS_SIGN))
		newBuf.Write(strconv.AppendInt([]byte{}, int64(countryCallingCode), 10))
		newBuf.WriteString(" ")
		newBuf.Write(formattedNumber.Bytes())
	case RFC3966:
		newBuf.WriteString(RFC3966_PREFIX)
		newBuf.WriteString(string(PLUS_SIGN))
		newBuf.Write(strconv.AppendInt([]byte{}, int64(countryCallingCode), 10))
		newBuf.WriteString("-")
		newBuf.Write(formattedNumber.Bytes())
	case NATIONAL:
	default:
	}
	formattedNumber.Reset()
	formattedNumber.Write(newBuf.Bytes())
}

// Simple wrapper of formatNsn for the common case of no carrier code.
func formatNsn(
	number string, metadata *PhoneMetadata, numberFormat PhoneNumberFormat) string {
	return formatNsnWithCarrier(number, metadata, numberFormat, "")
}

// Note in some regions, the national number can be written in two
// completely different ways depending on whether it forms part of the
// NATIONAL format or INTERNATIONAL format. The numberFormat parameter
// here is used to specify which format to use for those cases. If a
// carrierCode is specified, this will be inserted into the formatted
// string to replace $CC.
func formatNsnWithCarrier(
	number string,
	metadata *PhoneMetadata,
	numberFormat PhoneNumberFormat,
	carrierCode string) string {
	var intlNumberFormats []*NumberFormat = metadata.GetIntlNumberFormat()
	// When the intlNumberFormats exists, we use that to format national
	// number for the INTERNATIONAL format instead of using the
	// numberDesc.numberFormats.
	var availableFormats []*NumberFormat = metadata.GetIntlNumberFormat()
	if len(intlNumberFormats) == 0 || numberFormat == NATIONAL {
		availableFormats = metadata.GetNumberFormat()
	}
	var formattingPattern *NumberFormat = chooseFormattingPatternForNumber(
		availableFormats, number)
	if formattingPattern == nil {
		return number
	}
	return formatNsnUsingPatternWithCarrier(
		number, formattingPattern, numberFormat, carrierCode)
}

func chooseFormattingPatternForNumber(
	availableFormats []*NumberFormat,
	nationalNumber string) *NumberFormat {

	for _, numFormat := range availableFormats {
		leadingDigitsPattern := numFormat.GetLeadingDigitsPattern()
		size := len(leadingDigitsPattern)
		if size == 0 {
			continue
		}
		// We always use the last leading_digits_pattern, as it is the
		// most detailed.
		reg, ok := regexCache[leadingDigitsPattern[size-1]]
		if !ok {
			pat := leadingDigitsPattern[size-1]
			reg = regexp.MustCompile(pat)
			regexCache[pat] = reg
		}
		if reg.MatchString(nationalNumber) {
			return numFormat
		}
	}
	return nil
}

// Simple wrapper of formatNsnUsingPattern for the common case of no carrier code.
func formatNsnUsingPattern(
	nationalNumber string,
	formattingPattern *NumberFormat,
	numberFormat PhoneNumberFormat) string {
	return formatNsnUsingPatternWithCarrier(
		nationalNumber, formattingPattern, numberFormat, "")
}

// Note that carrierCode is optional - if null or an empty string, no
// carrier code replacement will take place.
func formatNsnUsingPatternWithCarrier(
	nationalNumber string,
	formattingPattern *NumberFormat,
	numberFormat PhoneNumberFormat,
	carrierCode string) string {

	numberFormatRule := formattingPattern.GetFormat()
	// TODO(ttacon): see what we should be doing if the pattern doesn't exist?
	m, ok := regexCache[formattingPattern.GetPattern()]
	if !ok {
		pat := formattingPattern.GetPattern()
		regexCache[pat] = regexp.MustCompile(pat)
		m = regexCache[pat]
	}

	formattedNationalNumber := ""
	if numberFormat == NATIONAL &&
		len(carrierCode) > 0 &&
		len(formattingPattern.GetDomesticCarrierCodeFormattingRule()) > 0 {
		// Replace the $CC in the formatting rule with the desired carrier code.
		carrierCodeFormattingRule := formattingPattern.GetDomesticCarrierCodeFormattingRule()
		carrierCodeFormattingRule =
			CC_PATTERN.ReplaceAllString(carrierCodeFormattingRule, carrierCode)
		// Now replace the $FG in the formatting rule with the first group
		// and the carrier code combined in the appropriate way.
		numberFormatRule = FIRST_GROUP_PATTERN.ReplaceAllString(
			numberFormatRule, carrierCodeFormattingRule)
		// TODO(ttacon): should these params be reversed?
		formattedNationalNumber = m.ReplaceAllString(numberFormatRule, nationalNumber)
	} else {
		// Use the national prefix formatting rule instead.
		nationalPrefixFormattingRule :=
			formattingPattern.GetNationalPrefixFormattingRule()
		if numberFormat == NATIONAL &&
			len(nationalPrefixFormattingRule) > 0 {

			formattedNationalNumber =
				m.ReplaceAllString(
					FIRST_GROUP_PATTERN.ReplaceAllString(
						numberFormatRule, nationalPrefixFormattingRule),
					nationalNumber)
		} else {
			formattedNationalNumber = m.ReplaceAllString(
				numberFormatRule, nationalNumber)
		}
	}
	if numberFormat == RFC3966 {
		// Strip any leading punctuation.
		if SEPARATOR_PATTERN.MatchString(formattedNationalNumber) {
			// TODO(ttacon): wrap *Regexp to have a function that
			// just matches the first occurence of something
			// (just make a closure that keeps track of when the
			// first match occurs)
		}
		// Replace the rest with a dash between each number group.
		// TODO(ttacon): this is unimplemented for now
	}
	return formattedNationalNumber
}

// Gets a valid number for the specified region.
func getExampleNumber(regionCode string) *PhoneNumber {
	return getExampleNumberForType(regionCode, FIXED_LINE)
}

// Gets a valid number for the specified region and number type.
func getExampleNumberForType(regionCode string, typ PhoneNumberType) *PhoneNumber {
	// Check the region code is valid.
	if !isValidRegionCode(regionCode) {
		glog.Warning("Invalid or unknown region code provided: " + regionCode)
		return nil
	}
	//PhoneNumberDesc (pointer?)
	var desc = getNumberDescByType(getMetadataForRegion(regionCode), typ)
	exNum := desc.GetExampleNumber()
	if len(exNum) > 0 {
		num, err := parse(exNum, regionCode)
		if err != nil {
			return nil
		}
		return num
	}
	return nil
}

// Gets a valid number for the specified country calling code for a
// non-geographical entity.
func getExampleNumberForNonGeoEntity(countryCallingCode int) *PhoneNumber {
	var metadata *PhoneMetadata = getMetadataForNonGeographicalRegion(countryCallingCode)
	if metadata == nil {
		return nil
	}
	var desc *PhoneNumberDesc = metadata.GetGeneralDesc()
	exNum := desc.GetExampleNumber()
	if len(exNum) > 0 {
		num, err := parse("+"+strconv.Itoa(countryCallingCode)+exNum, "ZZ")
		if err != nil {
			return nil
		}
		return num
	}
	return nil
}

// Appends the formatted extension of a phone number to formattedNumber,
// if the phone number had an extension specified.
func maybeAppendFormattedExtension(
	number *PhoneNumber,
	metadata *PhoneMetadata,
	numberFormat PhoneNumberFormat,
	formattedNumber *bytes.Buffer) {

	extension := number.GetExtension()
	if len(extension) == 0 {
		return
	}

	prefExtn := metadata.GetPreferredExtnPrefix()
	if numberFormat == RFC3966 {
		formattedNumber.WriteString(RFC3966_EXTN_PREFIX)
	} else if len(prefExtn) > 0 {
		formattedNumber.WriteString(prefExtn)
	} else {
		formattedNumber.WriteString(DEFAULT_EXTN_PREFIX)
	}
	formattedNumber.WriteString(extension)
}

func getNumberDescByType(
	metadata *PhoneMetadata,
	typ PhoneNumberType) *PhoneNumberDesc {

	switch typ {
	case PREMIUM_RATE:
		return metadata.GetPremiumRate()
	case TOLL_FREE:
		return metadata.GetTollFree()
	case MOBILE:
		return metadata.GetMobile()
	case FIXED_LINE:
		fallthrough
	case FIXED_LINE_OR_MOBILE:
		return metadata.GetFixedLine()
	case SHARED_COST:
		return metadata.GetSharedCost()
	case VOIP:
		return metadata.GetVoip()
	case PERSONAL_NUMBER:
		return metadata.GetPersonalNumber()
	case PAGER:
		return metadata.GetPager()
	case UAN:
		return metadata.GetUan()
	case VOICEMAIL:
		return metadata.GetVoicemail()
	default:
		return metadata.GetGeneralDesc()
	}
}

// Gets the type of a phone number.
func getNumberType(number *PhoneNumber) PhoneNumberType {
	var regionCode string = getRegionCodeForNumber(number)
	var metadata *PhoneMetadata = getMetadataForRegionOrCallingCode(
		int(number.GetCountryCode()), regionCode)
	if metadata == nil {
		return UNKNOWN
	}
	var nationalSignificantNumber = getNationalSignificantNumber(number)
	return getNumberTypeHelper(nationalSignificantNumber, metadata)
}

func getNumberTypeHelper(
	nationalNumber string,
	metadata *PhoneMetadata) PhoneNumberType {

	var generalNumberDesc *PhoneNumberDesc = metadata.GetGeneralDesc()
	var natNumPat = generalNumberDesc.GetNationalNumberPattern()
	if len(natNumPat) == 0 ||
		!isNumberMatchingDesc(nationalNumber, generalNumberDesc) {
		return UNKNOWN
	}

	if isNumberMatchingDesc(nationalNumber, metadata.GetPremiumRate()) {
		return PREMIUM_RATE
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetTollFree()) {
		return TOLL_FREE
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetSharedCost()) {
		return SHARED_COST
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetVoip()) {
		return VOIP
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetPersonalNumber()) {
		return PERSONAL_NUMBER
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetPager()) {
		return PAGER
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetUan()) {
		return UAN
	}
	if isNumberMatchingDesc(nationalNumber, metadata.GetVoicemail()) {
		return VOICEMAIL
	}

	var isFixedLine = isNumberMatchingDesc(
		nationalNumber, metadata.GetFixedLine())

	if isFixedLine {
		if metadata.GetSameMobileAndFixedLinePattern() {
			return FIXED_LINE_OR_MOBILE
		} else if isNumberMatchingDesc(nationalNumber, metadata.GetMobile()) {
			return FIXED_LINE_OR_MOBILE
		}
		return FIXED_LINE
	}
	// Otherwise, test to see if the number is mobile. Only do this if
	// certain that the patterns for mobile and fixed line aren't the same.
	if !metadata.GetSameMobileAndFixedLinePattern() &&
		isNumberMatchingDesc(nationalNumber, metadata.GetMobile()) {
		return MOBILE
	}
	return UNKNOWN
}

// Returns the metadata for the given region code or nil if the region
// code is invalid or unknown.
func getMetadataForRegion(regionCode string) *PhoneMetadata {
	if !isValidRegionCode(regionCode) {
		return nil
	}
	// TODO(ttacon): make this go rotine safe?
	// synchronized (regionToMetadataMap) {
	val, ok := regionToMetadataMap[regionCode]
	if !ok {
		// The regionCode here will be valid and won't be '001', so
		// we don't need to worry about what to pass in for the country
		// calling code.
		//loadMetadataFromFile(
		//	instance.currentFilePrefix, regionCode, 0, instance.metadataLoader)
	}
	return val
}

func getMetadataForNonGeographicalRegion(countryCallingCode int) *PhoneMetadata {
	// TODO(ttacon): make this go rotine safe?
	// synchronized (countryCodeToNonGeographicalMetadataMap) {
	_, ok := CountryCodeToRegion[countryCallingCode]
	if !ok {
		return nil
	}
	val, ok := countryCodeToNonGeographicalMetadataMap[countryCallingCode]
	if !ok {
		loadMetadataFromFile(
			instance.currentFilePrefix,
			REGION_CODE_FOR_NON_GEO_ENTITY,
			countryCallingCode,
			instance.metadataLoader)
	}
	return val
}

func isNumberPossibleForDesc(
	nationalNumber string, numberDesc *PhoneNumberDesc) bool {

	pat, ok := regexCache[numberDesc.GetPossibleNumberPattern()]
	if !ok {
		patP := numberDesc.GetPossibleNumberPattern()
		pat = regexp.MustCompile(patP)
		regexCache[patP] = pat
	}
	return pat.MatchString(nationalNumber)
}

func isNumberMatchingDesc(nationalNumber string, numberDesc *PhoneNumberDesc) bool {
	pat, ok := regexCache[numberDesc.GetNationalNumberPattern()]
	if !ok {
		patP := numberDesc.GetNationalNumberPattern()
		pat = regexp.MustCompile(patP)
		regexCache[patP] = pat
	}
	return isNumberPossibleForDesc(nationalNumber, numberDesc) &&
		pat.MatchString(nationalNumber)

}

// Tests whether a phone number matches a valid pattern. Note this doesn't
// verify the number is actually in use, which is impossible to tell by
// just looking at a number itself.
func isValidNumber(number *PhoneNumber) bool {
	var regionCode string = getRegionCodeForNumber(number)
	return isValidNumberForRegion(number, regionCode)
}

// Tests whether a phone number is valid for a certain region. Note this
// doesn't verify the number is actually in use, which is impossible to
// tell by just looking at a number itself. If the country calling code is
// not the same as the country calling code for the region, this immediately
// exits with false. After this, the specific number pattern rules for the
// region are examined. This is useful for determining for example whether
// a particular number is valid for Canada, rather than just a valid NANPA
// number.
// Warning: In most cases, you want to use isValidNumber() instead. For
// example, this method will mark numbers from British Crown dependencies
// such as the Isle of Man as invalid for the region "GB" (United Kingdom),
// since it has its own region code, "IM", which may be undesirable.
func isValidNumberForRegion(number *PhoneNumber, regionCode string) bool {
	var countryCode int = int(number.GetCountryCode())
	var metadata *PhoneMetadata = getMetadataForRegionOrCallingCode(
		countryCode, regionCode)
	if metadata == nil ||
		(REGION_CODE_FOR_NON_GEO_ENTITY != regionCode &&
			countryCode != GetCountryCodeForValidRegion(regionCode)) {
		// Either the region code was invalid, or the country calling
		// code for this number does not match that of the region code.
		return false
	}
	var generalNumDesc *PhoneNumberDesc = metadata.GetGeneralDesc()
	var nationalSignificantNumber string = getNationalSignificantNumber(number)
	// For regions where we don't have metadata for PhoneNumberDesc, we
	// treat any number passed in as a valid number if its national
	// significant number is between the minimum and maximum lengths
	// defined by ITU for a national significant number.
	if len(generalNumDesc.GetNationalNumberPattern()) == 0 {
		var numberLength int = len(nationalSignificantNumber)
		return numberLength > MIN_LENGTH_FOR_NSN &&
			numberLength <= MAX_LENGTH_FOR_NSN
	}
	return getNumberTypeHelper(nationalSignificantNumber, metadata) != UNKNOWN
}

// Returns the region where a phone number is from. This could be used for
// geocoding at the region level.
func getRegionCodeForNumber(number *PhoneNumber) string {
	var countryCode int = int(number.GetCountryCode())
	var regions []string = CountryCodeToRegion[countryCode]
	if len(regions) == 0 {
		var numberString string = getNationalSignificantNumber(number)
		glog.Warning(
			"Missing/invalid country_code (" +
				strconv.Itoa(countryCode) + ") for number " + numberString)
		return ""
	}
	if len(regions) == 1 {
		return regions[0]
	}
	return getRegionCodeForNumberFromRegionList(number, regions)
}

func getRegionCodeForNumberFromRegionList(
	number *PhoneNumber,
	regionCodes []string) string {

	var nationalNumber string = getNationalSignificantNumber(number)
	for _, regionCode := range regionCodes {
		// If leadingDigits is present, use this. Otherwise, do
		// full validation. Metadata cannot be null because the
		// region codes come from the country calling code map.
		var metadata *PhoneMetadata = getMetadataForRegion(regionCode)
		if len(metadata.GetLeadingDigits()) > 0 {
			// TODO(ttacon): is this okay? what error cases should we expect?
			pat, ok := regexCache[metadata.GetLeadingDigits()]
			if !ok {
				patP := metadata.GetLeadingDigits()
				pat = regexp.MustCompile(patP)
				regexCache[patP] = pat
			}
			if pat.MatchString(nationalNumber) {
				return regionCode
			}
		} else if getNumberTypeHelper(nationalNumber, metadata) != UNKNOWN {
			return regionCode
		}
	}
	return ""
}

// Returns the region code that matches the specific country calling code.
// In the case of no region code being found, ZZ will be returned. In the
// case of multiple regions, the one designated in the metadata as the
// "main" region for this calling code will be returned. If the
// countryCallingCode entered is valid but doesn't match a specific region
// (such as in the case of non-geographical calling codes like 800) the
// value "001" will be returned (corresponding to the value for World in
// the UN M.49 schema).
func getRegionCodeForCountryCode(countryCallingCode int) string {
	var regionCodes []string = CountryCodeToRegion[countryCallingCode]
	if len(regionCodes) == 0 {
		return UNKNOWN_REGION
	}
	return regionCodes[0]
}

// Returns a list with the region codes that match the specific country
// calling code. For non-geographical country calling codes, the region
// code 001 is returned. Also, in the case of no region code being found,
// an empty list is returned.
func getRegionCodesForCountryCode(countryCallingCode int) []string {
	var regionCodes []string = CountryCodeToRegion[countryCallingCode]
	return regionCodes
}

// Returns the country calling code for a specific region. For example, this
// would be 1 for the United States, and 64 for New Zealand.
func GetCountryCodeForRegion(regionCode string) int {
	if !isValidRegionCode(regionCode) {
		if len(regionCode) == 0 {
			regionCode = "null"
		}
		glog.Warning(
			"Invalid or missing region code (" + regionCode + ") provided.")
		return 0
	}
	return GetCountryCodeForValidRegion(regionCode)
}

// Returns the country calling code for a specific region. For example,
// this would be 1 for the United States, and 64 for New Zealand. Assumes
// the region is already valid.
func GetCountryCodeForValidRegion(regionCode string) int {
	var metadata *PhoneMetadata = getMetadataForRegion(regionCode)
	return int(metadata.GetCountryCode())
}

// Returns the national dialling prefix for a specific region. For example,
// this would be 1 for the United States, and 0 for New Zealand. Set
// stripNonDigits to true to strip symbols like "~" (which indicates a
// wait for a dialling tone) from the prefix returned. If no national prefix
// is present, we return null.
//
// Warning: Do not use this method for do-your-own formatting - for some
// regions, the national dialling prefix is used only for certain types
// of numbers. Use the library's formatting functions to prefix the
// national prefix when required.
func getNddPrefixForRegion(regionCode string, stripNonDigits bool) string {
	var metadata *PhoneMetadata = getMetadataForRegion(regionCode)
	if metadata == nil {
		if len(regionCode) == 0 {
			regionCode = "null"
		}
		glog.Warning(
			"Invalid or missing region code (" + regionCode + ") provided.")
		return ""
	}
	var nationalPrefix string = metadata.GetNationalPrefix()
	// If no national prefix was found, we return null.
	if len(nationalPrefix) == 0 {
		return ""
	}
	if stripNonDigits {
		// Note: if any other non-numeric symbols are ever used in
		// national prefixes, these would have to be removed here as well.
		nationalPrefix = strings.Replace(nationalPrefix, "~", "", -1)
	}
	return nationalPrefix
}

// Checks if this is a region under the North American Numbering Plan
// Administration (NANPA).
func isNANPACountry(regionCode string) bool {
	_, ok := nanpaRegions[regionCode]
	return ok
}

// Checks whether the country calling code is from a region whose national
// significant number could contain a leading zero. An example of such a
// region is Italy. Returns false if no metadata for the country is found.
func isLeadingZeroPossible(countryCallingCode int) bool {
	var mainMetadataForCallingCode *PhoneMetadata = getMetadataForRegionOrCallingCode(
		countryCallingCode,
		getRegionCodeForCountryCode(countryCallingCode),
	)
	return mainMetadataForCallingCode.GetLeadingZeroPossible()
}

// Checks if the number is a valid vanity (alpha) number such as 800
// MICROSOFT. A valid vanity number will start with at least 3 digits and
// will have three or more alpha characters. This does not do
// region-specific checks - to work out if this number is actually valid
// for a region, it should be parsed and methods such as
// isPossibleNumberWithReason() and isValidNumber() should be used.
func isAlphaNumber(number string) bool {
	if !isViablePhoneNumber(number) {
		// Number is too short, or doesn't match the basic phone
		// number pattern.
		return false
	}
	strippedNumber := bytes.NewBufferString(number)
	maybeStripExtension(strippedNumber)
	return VALID_ALPHA_PHONE_PATTERN.MatchString(strippedNumber.String())
}

// Convenience wrapper around isPossibleNumberWithReason(). Instead of
// returning the reason for failure, this method returns a boolean value.
func isPossibleNumber(number *PhoneNumber) bool {
	return isPossibleNumberWithReason(number) == IS_POSSIBLE
}

// Helper method to check a number against a particular pattern and
// determine whether it matches, or is too short or too long. Currently,
// if a number pattern suggests that numbers of length 7 and 10 are
// possible, and a number in between these possible lengths is entered,
// such as of length 8, this will return TOO_LONG.
func testNumberLengthAgainstPattern(
	numberPattern *regexp.Regexp,
	number string) ValidationResult {

	if numberPattern.MatchString(number) {
		return IS_POSSIBLE
	}

	// TODO(ttacon): we need some similar functionality to lookingAt()
	//if (numberMatcher.lookingAt()) {
	//  return ValidationResult.TOO_LONG;
	//} else {
	//  return ValidationResult.TOO_SHORT;
	//}
	return TOO_SHORT
}

// Helper method to check whether a number is too short to be a regular
// length phone number in a region.
func isShorterThanPossibleNormalNumber(
	regionMetadata *PhoneMetadata,
	number string) bool {

	pat, ok := regexCache[regionMetadata.GetGeneralDesc().GetPossibleNumberPattern()]
	if !ok {
		patP := regionMetadata.GetGeneralDesc().GetPossibleNumberPattern()
		pat = regexp.MustCompile(patP)
		regexCache[patP] = pat
	}
	return testNumberLengthAgainstPattern(pat, number) == TOO_SHORT
}

// Check whether a phone number is a possible number. It provides a more
// lenient check than isValidNumber() in the following sense:
//
//  - It only checks the length of phone numbers. In particular, it
//    doesn't check starting digits of the number.
//  - It doesn't attempt to figure out the type of the number, but uses
//    general rules which applies to all types of phone numbers in a
//    region. Therefore, it is much faster than isValidNumber.
//  - For fixed line numbers, many regions have the concept of area code,
//    which together with subscriber number constitute the national
//    significant number. It is sometimes okay to dial the subscriber number
//    only when dialing in the same area. This function will return true
//    if the subscriber-number-only version is passed in. On the other hand,
//    because isValidNumber validates using information on both starting
//    digits (for fixed line numbers, that would most likely be area codes)
//    and length (obviously includes the length of area codes for fixed
//    line numbers), it will return false for the subscriber-number-only
//    version.
func isPossibleNumberWithReason(number *PhoneNumber) ValidationResult {
	nationalNumber := getNationalSignificantNumber(number)
	countryCode := int(number.GetCountryCode())
	// Note: For Russian Fed and NANPA numbers, we just use the rules
	// from the default region (US or Russia) since the
	// getRegionCodeForNumber will not work if the number is possible
	// but not valid. This would need to be revisited if the possible
	// number pattern ever differed between various regions within
	// those plans.
	if !hasValidCountryCallingCode(countryCode) {
		return INVALID_COUNTRY_CODE
	}
	regionCode := getRegionCodeForCountryCode(countryCode)
	// Metadata cannot be null because the country calling code is valid.
	var metadata *PhoneMetadata = getMetadataForRegionOrCallingCode(
		countryCode, regionCode)
	var generalNumDesc *PhoneNumberDesc = metadata.GetGeneralDesc()
	// Handling case of numbers with no metadata.
	if len(generalNumDesc.GetNationalNumberPattern()) == 0 {
		glog.Info("Checking if number is possible with incomplete metadata.")
		numberLength := len(nationalNumber)
		if numberLength < MIN_LENGTH_FOR_NSN {
			return TOO_SHORT
		} else if numberLength > MAX_LENGTH_FOR_NSN {
			return TOO_LONG
		} else {
			return IS_POSSIBLE
		}
	}
	pat, ok := regexCache[generalNumDesc.GetPossibleNumberPattern()]
	if !ok {
		patP := generalNumDesc.GetPossibleNumberPattern()
		regexCache[patP] = regexp.MustCompile(patP)
		pat = regexCache[patP]
	}
	return testNumberLengthAgainstPattern(pat, nationalNumber)
}

// Check whether a phone number is a possible number given a number in the
// form of a string, and the region where the number could be dialed from.
// It provides a more lenient check than isValidNumber(). See
// isPossibleNumber(PhoneNumber) for details.
//
// This method first parses the number, then invokes
// isPossibleNumber(PhoneNumber) with the resultant PhoneNumber object.
func isPossibleNumberWithRegion(number, regionDialingFrom string) bool {
	num, err := parse(number, regionDialingFrom)
	if err != nil {
		return false
	}
	return isPossibleNumber(num)
}

// Attempts to extract a valid number from a phone number that is too long
// to be valid, and resets the PhoneNumber object passed in to that valid
// version. If no valid number could be extracted, the PhoneNumber object
// passed in will not be modified.
func truncateTooLongNumber(number *PhoneNumber) bool {
	if isValidNumber(number) {
		return true
	}
	var numberCopy *PhoneNumber
	proto.Merge(numberCopy, number)
	nationalNumber := number.GetNationalNumber()
	nationalNumber /= 10
	numberCopy.NationalNumber = proto.Uint64(nationalNumber)
	if isPossibleNumberWithReason(numberCopy) == TOO_SHORT || nationalNumber == 0 {
		return false
	}
	for !isValidNumber(numberCopy) {
		nationalNumber /= 10
		numberCopy.NationalNumber = proto.Uint64(nationalNumber)
		if isPossibleNumberWithReason(numberCopy) == TOO_SHORT ||
			nationalNumber == 0 {
			return false
		}
	}

	number.NationalNumber = proto.Uint64(nationalNumber)
	return true
}

// Gets an AsYouTypeFormatter for the specific region.
// TODO(ttacon): uncomment once we do asyoutypeformatter.go
//public AsYouTypeFormatter getAsYouTypeFormatter(String regionCode) {
//    return new AsYouTypeFormatter(regionCode);
//}

// Extracts country calling code from fullNumber, returns it and places
// the remaining number in nationalNumber. It assumes that the leading plus
// sign or IDD has already been removed. Returns 0 if fullNumber doesn't
// start with a valid country calling code, and leaves nationalNumber
// unmodified.
func extractCountryCode(fullNumber, nationalNumber *bytes.Buffer) int {
	// TODO(ttacon): clean all this up when replace *bytes.Buffer usage
	// with insertablebuffer.Buf
	if len(fullNumber.Bytes()) == 0 || (fullNumber.Bytes()[0] == '0') {
		// Country codes do not begin with a '0'.
		return 0
	}
	var potentialCountryCode int
	var numberLength = len(fullNumber.String())
	for i := 1; i <= MAX_LENGTH_COUNTRY_CODE && i <= numberLength; i++ {
		potentialCountryCode, _ = strconv.Atoi(fullNumber.String()[0:i])
		if _, ok := CountryCodeToRegion[potentialCountryCode]; ok {
			nationalNumber.WriteString(fullNumber.String()[i:])
			return potentialCountryCode
		}
	}
	return 0
}

// Tries to extract a country calling code from a number. This method will
// return zero if no country calling code is considered to be present.
// Country calling codes are extracted in the following ways:
//
//  - by stripping the international dialing prefix of the region the
//    person is dialing from, if this is present in the number, and looking
//    at the next digits
//  - by stripping the '+' sign if present and then looking at the next digits
//  - by comparing the start of the number and the country calling code of
//    the default region. If the number is not considered possible for the
//    numbering plan of the default region initially, but starts with the
//    country calling code of this region, validation will be reattempted
//    after stripping this country calling code. If this number is considered a
//    possible number, then the first digits will be considered the country
//    calling code and removed as such.
//
// It will throw a NumberParseException if the number starts with a '+' but
// the country calling code supplied after this does not match that of any
// known region.
// @VisibleForTesting
func maybeExtractCountryCode(
	number string,
	defaultRegionMetadata *PhoneMetadata,
	nationalNumber *bytes.Buffer,
	keepRawInput bool,
	phoneNumber *PhoneNumber) int {

	if len(number) == 0 {
		return 0
	}
	fullNumber := bytes.NewBufferString(number)
	// Set the default prefix to be something that will never match.
	possibleCountryIddPrefix := "NonMatch"
	if defaultRegionMetadata != nil {
		possibleCountryIddPrefix = defaultRegionMetadata.GetInternationalPrefix()
	}

	countryCodeSource :=
		maybeStripInternationalPrefixAndNormalize(fullNumber, possibleCountryIddPrefix)
	if keepRawInput {
		phoneNumber.CountryCodeSource = &countryCodeSource
	}
	if countryCodeSource != PhoneNumber_FROM_DEFAULT_COUNTRY {
		if len(fullNumber.String()) <= MIN_LENGTH_FOR_NSN {
			// TODO(ttacon): remove this when replacing with idiomatic go
			panic(
				"Phone number had an IDD, but after this was not " +
					"long enough to be a viable phone number.")
		}
		potentialCountryCode := extractCountryCode(fullNumber, nationalNumber)
		if potentialCountryCode != 0 {
			phoneNumber.CountryCode = proto.Int(potentialCountryCode)
			return potentialCountryCode
		}

		// If this fails, they must be using a strange country calling code
		// that we don't recognize, or that doesn't exist.
		panic("Country calling code supplied was not recognised.")
	} else if defaultRegionMetadata != nil {
		// Check to see if the number starts with the country calling code
		// for the default region. If so, we remove the country calling
		// code, and do some checks on the validity of the number before
		// and after.
		defaultCountryCode := int(defaultRegionMetadata.GetCountryCode())
		defaultCountryCodeString := strconv.Itoa(defaultCountryCode)
		normalizedNumber := fullNumber.String()
		if strings.HasPrefix(normalizedNumber, defaultCountryCodeString) {
			potentialNationalNumber := bytes.NewBufferString(
				normalizedNumber[len(defaultCountryCodeString):])
			var generalDesc *PhoneNumberDesc = defaultRegionMetadata.GetGeneralDesc()
			validNumberPattern, ok := regexCache[generalDesc.GetNationalNumberPattern()]
			if !ok {
				pat := generalDesc.GetNationalNumberPattern()
				validNumberPattern = regexp.MustCompile(pat)
				regexCache[pat] = validNumberPattern
			}
			maybeStripNationalPrefixAndCarrierCode(
				potentialNationalNumber,
				defaultRegionMetadata,
				bytes.NewBuffer(nil) /* Don't need the carrier code */)
			possibleNumberPattern, ok := regexCache[generalDesc.GetPossibleNumberPattern()]
			if !ok {
				pat := generalDesc.GetPossibleNumberPattern()
				possibleNumberPattern = regexp.MustCompile(pat)
				regexCache[pat] = possibleNumberPattern
			}
			// If the number was not valid before but is valid now, or
			// if it was too long before, we consider the number with
			// the country calling code stripped to be a better result and
			// keep that instead.
			if (!validNumberPattern.MatchString(fullNumber.String()) &&
				validNumberPattern.MatchString(potentialNationalNumber.String())) ||
				testNumberLengthAgainstPattern(
					possibleNumberPattern, fullNumber.String()) == TOO_LONG {
				nationalNumber.Write(potentialNationalNumber.Bytes())
				if keepRawInput {
					val := PhoneNumber_FROM_NUMBER_WITHOUT_PLUS_SIGN
					phoneNumber.CountryCodeSource = &val
				}
				phoneNumber.CountryCode = proto.Int(defaultCountryCode)
				return defaultCountryCode
			}
		}
	}
	// No country calling code present.
	phoneNumber.CountryCode = proto.Int(0)
	return 0
}

// Strips the IDD from the start of the number if present. Helper function
// used by maybeStripInternationalPrefixAndNormalize.
func parsePrefixAsIdd(iddPattern *regexp.Regexp, number *bytes.Buffer) bool {
	//if (m.lookingAt()) {
	ind := iddPattern.FindIndex(number.Bytes())
	if len(ind) == 0 {
		return false
	}
	matchEnd := ind[1] // ind is a two element slice
	// Only strip this if the first digit after the match is not
	// a 0, since country calling codes cannot begin with 0.
	find := CAPTURING_DIGIT_PATTERN.FindAllString(number.String()[matchEnd:], -1)
	if len(find) > 0 {
		normalizedGroup := normalizeDigitsOnly(find[1])
		if normalizedGroup == "0" {
			return false
		}
	}

	by := make([]byte, len(number.Bytes())-matchEnd)
	copy(by, number.Bytes()[matchEnd:])
	number.Reset()
	number.Write(by)
	return true
}

// Strips any international prefix (such as +, 00, 011) present in the
// number provided, normalizes the resulting number, and indicates if
// an international prefix was present.
// @VisibleForTesting
func maybeStripInternationalPrefixAndNormalize(
	number *bytes.Buffer,
	possibleIddPrefix string) PhoneNumber_CountryCodeSource {

	if len(number.String()) == 0 {
		return PhoneNumber_FROM_DEFAULT_COUNTRY
	}
	// Check to see if the number begins with one or more plus signs.
	ind := PLUS_CHARS_PATTERN.FindAllIndex(number.Bytes(), -1)
	if len(ind) > 0 {
		last := ind[len(ind)-1][1]
		numBytes := make([]byte, len(number.Bytes())-last)
		copy(numBytes, number.Bytes()[last:])
		number.Reset()
		number.Write(numBytes)
		// Can now normalize the rest of the number since we've consumed
		// the "+" sign at the start.
		val := normalize(number.String())
		number.Reset()
		number.WriteString(val)
		return PhoneNumber_FROM_NUMBER_WITH_PLUS_SIGN
	}

	// Attempt to parse the first digits as an international prefix.
	iddPattern, ok := regexCache[possibleIddPrefix]
	if !ok {
		pat := possibleIddPrefix
		iddPattern = regexp.MustCompile(pat)
		regexCache[pat] = iddPattern
	}
	val := normalize(number.String())
	number.Reset()
	number.WriteString(val)
	if parsePrefixAsIdd(iddPattern, number) {
		return PhoneNumber_FROM_NUMBER_WITH_IDD
	}
	return PhoneNumber_FROM_DEFAULT_COUNTRY
}

// Strips any national prefix (such as 0, 1) present in the number provided.
// @VisibleForTesting
func maybeStripNationalPrefixAndCarrierCode(
	number *bytes.Buffer,
	metadata *PhoneMetadata,
	carrierCode *bytes.Buffer) bool {

	numberLength := len(number.String())
	possibleNationalPrefix := metadata.GetNationalPrefixForParsing()
	if numberLength == 0 || len(possibleNationalPrefix) == 0 {
		// Early return for numbers of zero length.
		return false
	}
	// Attempt to parse the first digits as a national prefix.
	prefixMatcher, ok := regexCache[possibleNationalPrefix]
	if !ok {
		pat := possibleNationalPrefix
		prefixMatcher = regexp.MustCompile(pat)
		regexCache[pat] = prefixMatcher
	}
	if prefixMatcher.Match(number.Bytes()) {
		//if (prefixMatcher.lookingAt()) {
		nationalNumberRule, ok :=
			regexCache[metadata.GetGeneralDesc().GetNationalNumberPattern()]
		if !ok {
			pat := metadata.GetGeneralDesc().GetNationalNumberPattern()
			nationalNumberRule = regexp.MustCompile(pat)
			regexCache[pat] = nationalNumberRule
		}
		// Check if the original number is viable.
		isViableOriginalNumber := nationalNumberRule.Match(number.Bytes())
		// prefixMatcher.group(numOfGroups) == null implies nothing was
		// captured by the capturing groups in possibleNationalPrefix;
		// therefore, no transformation is necessary, and we just
		// remove the national prefix.
		groups := prefixMatcher.FindAllIndex(number.Bytes(), -1)
		numOfGroups := len(groups)
		transformRule := metadata.GetNationalPrefixTransformRule()
		if len(transformRule) == 0 || len(groups[numOfGroups-1]) == 0 {
			// If the original number was viable, and the resultant number
			// is not, we return.
			if isViableOriginalNumber &&
				!nationalNumberRule.MatchString(
					number.String()[groups[len(groups)-1][1]:]) {
				return false
			}
			if len(carrierCode.Bytes()) != 0 &&
				numOfGroups > 0 &&
				len(groups[numOfGroups-1]) != 0 {
				carrierCode.Write(number.Bytes()[groups[0][0]:groups[0][1]])
			}
			numBytes := make([]byte, len(number.Bytes())-groups[0][1])
			copy(numBytes, number.Bytes()[groups[0][1]:])
			number.Reset()
			number.Write(numBytes)
			return true
		} else {
			// Check that the resultant number is still viable. If not,
			// return. Check this by copying the string buffer and
			// making the transformation on the copy first.
			transformedNumber := bytes.NewBuffer(number.Bytes())
			transformedNumBytes := number.Bytes()
			copy(transformedNumBytes[0:numberLength],
				prefixMatcher.ReplaceAllString(number.String(), transformRule))
			// v-- so confused...?
			//transformedNumber.replace(0, numberLength, prefixMatcher.replaceFirst(transformRule));
			if isViableOriginalNumber &&
				!nationalNumberRule.Match(transformedNumBytes) {
				return false
			}
			if len(carrierCode.Bytes()) != 0 && numOfGroups > 1 {
				carrierCode.WriteString(prefixMatcher.FindString(number.String()))
			}
			number.Reset()
			number.Write(transformedNumber.Bytes())
			return true
		}
	}
	return false
}

// Strips any extension (as in, the part of the number dialled after the
// call is connected, usually indicated with extn, ext, x or similar) from
// the end of the number, and returns it.
// @VisibleForTesting
func maybeStripExtension(number *bytes.Buffer) string {
	// If we find a potential extension, and the number preceding this is
	// a viable number, we assume it is an extension.
	ind := EXTN_PATTERN.FindIndex(number.Bytes())
	if len(ind) > 0 && isViablePhoneNumber(number.String()[0:ind[0]]) {
		// The numbers are captured into groups in the regular expression.
		for _, extensionGroup := range EXTN_PATTERN.FindAllIndex(number.Bytes(), -1) {
			if len(extensionGroup) == 0 {
				continue
			}
			// We go through the capturing groups until we find one
			// that captured some digits. If none did, then we will
			// return the empty string.
			extension := number.String()[extensionGroup[0]:extensionGroup[1]]
			numBytes := number.Bytes()
			number.Reset()
			number.Write(numBytes[0:ind[0]])
			return extension
		}
	}
	return ""
}

// Checks to see that the region code used is valid, or if it is not valid,
// that the number to parse starts with a + symbol so that we can attempt
// to infer the region from the number. Returns false if it cannot use the
// region provided and the region cannot be inferred.
func checkRegionForParsing(numberToParse, defaultRegion string) bool {
	if !isValidRegionCode(defaultRegion) {
		// If the number is null or empty, we can't infer the region.
		if len(numberToParse) == 0 ||
			!PLUS_CHARS_PATTERN.MatchString(numberToParse) {
			return false
		}
	}
	return true
}

// Parses a string and returns it in proto buffer format. This method will
// throw a NumberParseException if the number is not considered to be a
// possible number. Note that validation of whether the number is actually
// a valid number for a particular region is not performed. This can be
// done separately with isValidNumber().
func parse(numberToParse, defaultRegion string) (*PhoneNumber, error) {
	var phoneNumber *PhoneNumber = &PhoneNumber{}
	err := parseToNumber(numberToParse, defaultRegion, phoneNumber)
	return phoneNumber, err
}

// Same as parse(string, string), but accepts mutable PhoneNumber as a
// parameter to decrease object creation when invoked many times.
func parseToNumber(numberToParse, defaultRegion string, phoneNumber *PhoneNumber) error {
	return parseHelper(numberToParse, defaultRegion, false, true, phoneNumber)
}

// Parses a string and returns it in proto buffer format. This method
// differs from parse() in that it always populates the raw_input field of
// the protocol buffer with numberToParse as well as the country_code_source
// field.
func parseAndKeepRawInput(
	numberToParse, defaultRegion string) (*PhoneNumber, error) {
	var phoneNumber *PhoneNumber = &PhoneNumber{}
	err := parseAndKeepRawInputToNumber(numberToParse, defaultRegion, phoneNumber)
	return phoneNumber, err
}

// Same as parseAndKeepRawInput(String, String), but accepts a mutable
// PhoneNumber as a parameter to decrease object creation when invoked many
// times.
func parseAndKeepRawInputToNumber(
	numberToParse, defaultRegion string,
	phoneNumber *PhoneNumber) error {
	return parseHelper(numberToParse, defaultRegion, true, true, phoneNumber)
}

// Returns an iterable over all PhoneNumberMatch PhoneNumberMatches in text.
// This is a shortcut for findNumbers(CharSequence, String, Leniency, long)
// getMatcher(text, defaultRegion, Leniency.VALID, Long.MAX_VALUE)}.
//public Iterable<PhoneNumberMatch> findNumbers(CharSequence text, String defaultRegion) {
//    return findNumbers(text, defaultRegion, Leniency.VALID, Long.MAX_VALUE);
//}

// Returns an iterable over all PhoneNumberMatch PhoneNumberMatches in text.
//public Iterable<PhoneNumberMatch> findNumbers(
//	final CharSequence text, final String defaultRegion, final Leniency leniency,
//	final long maxTries) {
//
//		return new Iterable<PhoneNumberMatch>() {
//			public Iterator<PhoneNumberMatch> iterator() {
//				return new PhoneNumberMatcher(
//					PhoneNumberUtil.this, text, defaultRegion, leniency, maxTries);
//			}
//		};
//	}

// A helper function to set the values related to leading zeros in a
// PhoneNumber.
func setItalianLeadingZerosForPhoneNumber(
	nationalNumber string, phoneNumber *PhoneNumber) {
	if len(nationalNumber) > 1 && nationalNumber[0] == '0' {
		phoneNumber.ItalianLeadingZero = proto.Bool(true)
		numberOfLeadingZeros := 1
		// Note that if the national number is all "0"s, the last "0"
		// is not counted as a leading zero.

		for numberOfLeadingZeros < len(nationalNumber)-1 &&
			nationalNumber[numberOfLeadingZeros] == '0' {
			numberOfLeadingZeros++
		}
		if numberOfLeadingZeros != 1 {
			phoneNumber.NumberOfLeadingZeros = proto.Int(numberOfLeadingZeros)
		}
	}
}

var ErrInvalidCountryCode = errors.New("invalid country code")

// Parses a string and fills up the phoneNumber. This method is the same
// as the public parse() method, with the exception that it allows the
// default region to be null, for use by isNumberMatch(). checkRegion should
// be set to false if it is permitted for the default region to be null or
// unknown ("ZZ").
func parseHelper(
	numberToParse, defaultRegion string,
	keepRawInput, checkRegion bool,
	phoneNumber *PhoneNumber) error {
	if len(numberToParse) == 0 {
		return errors.New("The phone number supplied was empty.")
	} else if len(numberToParse) > MAX_INPUT_STRING_LENGTH {
		return errors.New("The string supplied was too long to parse.")
	}

	nationalNumber := bytes.NewBuffer(nil)
	buildNationalNumberForParsing(numberToParse, nationalNumber)

	if !isViablePhoneNumber(nationalNumber.String()) {
		return errors.New("The string supplied did not seem to be a phone number.")
	}

	// Check the region supplied is valid, or that the extracted number
	// starts with some sort of + sign so the number's region can be determined.
	if checkRegion &&
		!checkRegionForParsing(nationalNumber.String(), defaultRegion) {
		return errors.New("Missing or invalid default region.")
	}

	if keepRawInput {
		phoneNumber.RawInput = proto.String(numberToParse)
	}
	// Attempt to parse extension first, since it doesn't require
	// region-specific data and we want to have the non-normalised
	// number here.
	extension := maybeStripExtension(nationalNumber)
	if len(extension) > 0 {
		phoneNumber.Extension = proto.String(extension)
	}
	var regionMetadata *PhoneMetadata = getMetadataForRegion(defaultRegion)
	// Check to see if the number is given in international format so we
	// know whether this number is from the default region or not.
	normalizedNationalNumber := bytes.NewBuffer(nil)
	var countryCode = 0
	// TODO: This method should really just take in the string buffer that
	// has already been created, and just remove the prefix, rather than
	// taking in a string and then outputting a string buffer.
	countryCode = maybeExtractCountryCode(nationalNumber.String(), regionMetadata,
		normalizedNationalNumber, keepRawInput, phoneNumber)
	// TODO(ttacon): add error to above method so we can deal with plus
	//    } catch (NumberParseException e) {
	//		Matcher matcher = PLUS_CHARS_PATTERN.matcher(nationalNumber.toString());
	//		if (e.getErrorType() == NumberParseException.ErrorType.INVALID_COUNTRY_CODE &&
	//			matcher.lookingAt()) {
	///			// Strip the plus-char, and try again.
	//			countryCode = maybeExtractCountryCode(nationalNumber.substring(matcher.end()),
	//				regionMetadata, normalizedNationalNumber,
	//				keepRawInput, phoneNumber);
	//			if (countryCode == 0) {
	//				throw new NumberParseException(NumberParseException.ErrorType.INVALID_COUNTRY_CODE,
	//					"Could not interpret numbers after plus-sign.");
	//			}
	//		} else {
	//			throw new NumberParseException(e.getErrorType(), e.getMessage());
	//		}
	//    }
	if countryCode != 0 {
		phoneNumberRegion := getRegionCodeForCountryCode(countryCode)
		if phoneNumberRegion != defaultRegion {
			// Metadata cannot be null because the country calling
			// code is valid.
			regionMetadata = getMetadataForRegionOrCallingCode(
				countryCode, phoneNumberRegion)
		}
	} else {
		// If no extracted country calling code, use the region supplied
		// instead. The national number is just the normalized version of
		// the number we were given to parse.
		val := normalize(nationalNumber.String())
		normalizedNationalNumber.WriteString(val)
		if len(defaultRegion) != 0 {
			countryCode = int(regionMetadata.GetCountryCode())
			phoneNumber.CountryCode = proto.Int(countryCode)
		} else if keepRawInput {
			phoneNumber.CountryCodeSource = nil
		}
	}
	if len(normalizedNationalNumber.String()) < MIN_LENGTH_FOR_NSN {
		return errors.New("The string supplied is too short to be a phone number.")
	}
	if regionMetadata != nil {
		carrierCode := bytes.NewBuffer(nil)
		potentialNationalNumber := bytes.NewBuffer(normalizedNationalNumber.Bytes())
		maybeStripNationalPrefixAndCarrierCode(
			potentialNationalNumber, regionMetadata, carrierCode)
		// We require that the NSN remaining after stripping the national
		// prefix and carrier code be of a possible length for the region.
		// Otherwise, we don't do the stripping, since the original number
		// could be a valid short number.
		if !isShorterThanPossibleNormalNumber(
			regionMetadata, potentialNationalNumber.String()) {
			normalizedNationalNumber = potentialNationalNumber
			if keepRawInput {
				phoneNumber.PreferredDomesticCarrierCode =
					proto.String(carrierCode.String())
			}
		}
	}
	lengthOfNationalNumber := len(normalizedNationalNumber.String())
	if lengthOfNationalNumber < MIN_LENGTH_FOR_NSN {
		return errors.New("The string supplied is too short to be a phone number.")
	}
	if lengthOfNationalNumber > MAX_LENGTH_FOR_NSN {
		return ErrNumTooLong
	}
	setItalianLeadingZerosForPhoneNumber(
		normalizedNationalNumber.String(), phoneNumber)
	val, _ := strconv.ParseUint(normalizedNationalNumber.String(), 10, 64)
	phoneNumber.NationalNumber = proto.Uint64(val)
	return nil
}

var ErrNumTooLong = errors.New("The string supplied is too long to be a phone number.")

// Converts numberToParse to a form that we can parse and write it to
// nationalNumber if it is written in RFC3966; otherwise extract a possible
// number out of it and write to nationalNumber.
func buildNationalNumberForParsing(
	numberToParse string,
	nationalNumber *bytes.Buffer) {

	indexOfPhoneContext := strings.Index(numberToParse, RFC3966_PHONE_CONTEXT)
	if indexOfPhoneContext > 0 {
		phoneContextStart := indexOfPhoneContext + len(RFC3966_PHONE_CONTEXT)
		// If the phone context contains a phone number prefix, we need
		// to capture it, whereas domains will be ignored.
		if numberToParse[phoneContextStart] == PLUS_SIGN {
			// Additional parameters might follow the phone context. If so,
			// we will remove them here because the parameters after phone
			// context are not important for parsing the phone number.
			phoneContextEnd := strings.Index(numberToParse[phoneContextStart:], ";")
			if phoneContextEnd > 0 {
				nationalNumber.WriteString(
					numberToParse[phoneContextStart:phoneContextEnd])
			} else {
				nationalNumber.WriteString(numberToParse[phoneContextStart:])
			}
		}
		// Now append everything between the "tel:" prefix and the
		// phone-context. This should include the national number, an
		// optional extension or isdn-subaddress component. Note we also
		// handle the case when "tel:" is missing, as we have seen in some
		// of the phone number inputs. In that case, we append everything
		// from the beginning.
		indexOfRfc3966Prefix := strings.Index(numberToParse, RFC3966_PREFIX)
		indexOfNationalNumber := 0
		if indexOfRfc3966Prefix >= 0 {
			indexOfNationalNumber = indexOfRfc3966Prefix + len(RFC3966_PREFIX)
		}
		nationalNumber.WriteString(
			numberToParse[indexOfNationalNumber:indexOfPhoneContext])
	} else {
		// Extract a possible number from the string passed in (this
		// strips leading characters that could not be the start of a
		// phone number.)
		nationalNumber.WriteString(extractPossibleNumber(numberToParse))
	}

	// Delete the isdn-subaddress and everything after it if it is present.
	// Note extension won't appear at the same time with isdn-subaddress
	// according to paragraph 5.3 of the RFC3966 spec,
	indexOfIsdn := strings.Index(nationalNumber.String(), RFC3966_ISDN_SUBADDRESS)
	if indexOfIsdn > 0 {
		natNumBytes := nationalNumber.Bytes()
		nationalNumber.Reset()
		nationalNumber.Write(natNumBytes[:indexOfIsdn])
	}
	// If both phone context and isdn-subaddress are absent but other
	// parameters are present, the parameters are left in nationalNumber.
	// This is because we are concerned about deleting content from a
	// potential number string when there is no strong evidence that the
	// number is actually written in RFC3966.
}

// Takes two phone numbers and compares them for equality.
//
// Returns EXACT_MATCH if the country_code, NSN, presence of a leading zero
// for Italian numbers and any extension present are the same.
// Returns NSN_MATCH if either or both has no region specified, and the NSNs
// and extensions are the same.
// Returns SHORT_NSN_MATCH if either or both has no region specified, or the
// region specified is the same, and one NSN could be a shorter version of
// the other number. This includes the case where one has an extension
// specified, and the other does not.
// Returns NO_MATCH otherwise.
// For example, the numbers +1 345 657 1234 and 657 1234 are a SHORT_NSN_MATCH.
// The numbers +1 345 657 1234 and 345 657 are a NO_MATCH.
func isNumberMatchWithNumbers(firstNumberIn, secondNumberIn *PhoneNumber) MatchType {
	// Make copies of the phone number so that the numbers passed in are not edited.
	var firstNumber, secondNumber *PhoneNumber
	proto.Merge(firstNumber, firstNumberIn)
	proto.Merge(secondNumber, secondNumberIn)
	// First clear raw_input, country_code_source and
	// preferred_domestic_carrier_code fields and any empty-string
	// extensions so that we can use the proto-buffer equality method.
	firstNumber.RawInput = nil
	firstNumber.CountryCodeSource = nil
	firstNumber.PreferredDomesticCarrierCode = nil
	secondNumber.RawInput = nil
	secondNumber.CountryCodeSource = nil
	secondNumber.PreferredDomesticCarrierCode = nil

	firstNumExt := firstNumber.GetExtension()
	secondNumExt := secondNumber.GetExtension()
	// NOTE(ttacon): don't think we need this in go land...
	if len(firstNumExt) == 0 {
		firstNumber.Extension = nil
	}
	if len(secondNumExt) == 0 {
		secondNumber.Extension = nil
	}

	// Early exit if both had extensions and these are different.
	if len(firstNumExt) > 0 && len(secondNumExt) > 0 &&
		firstNumExt != secondNumExt {
		return NO_MATCH
	}
	var (
		firstNumberCountryCode  = firstNumber.GetCountryCode()
		secondNumberCountryCode = secondNumber.GetCountryCode()
	)
	// Both had country_code specified.
	if firstNumberCountryCode != 0 && secondNumberCountryCode != 0 {
		// TODO(ttacon): remove when make gen-equals
		if reflect.DeepEqual(firstNumber, secondNumber) {
			return EXACT_MATCH
		} else if firstNumberCountryCode == secondNumberCountryCode &&
			isNationalNumberSuffixOfTheOther(firstNumber, secondNumber) {
			// A SHORT_NSN_MATCH occurs if there is a difference because of
			// the presence or absence of an 'Italian leading zero', the
			// presence or absence of an extension, or one NSN being a
			// shorter variant of the other.
			return SHORT_NSN_MATCH
		}
		// This is not a match.
		return NO_MATCH
	}
	// Checks cases where one or both country_code fields were not
	// specified. To make equality checks easier, we first set the
	// country_code fields to be equal.
	firstNumber.CountryCode = proto.Int(int(secondNumberCountryCode))
	// If all else was the same, then this is an NSN_MATCH.
	// TODO(ttacon): remove when make gen-equals
	if reflect.DeepEqual(firstNumber, secondNumber) {
		return NSN_MATCH
	}
	if isNationalNumberSuffixOfTheOther(firstNumber, secondNumber) {
		return SHORT_NSN_MATCH
	}
	return NO_MATCH
}

// Returns true when one national number is the suffix of the other or both
// are the same.
func isNationalNumberSuffixOfTheOther(firstNumber, secondNumber *PhoneNumber) bool {
	var (
		firstNumberNationalNumber = strconv.FormatUint(
			firstNumber.GetNationalNumber(), 10)
		secondNumberNationalNumber = strconv.FormatUint(
			secondNumber.GetNationalNumber(), 10)
	)
	// Note that endsWith returns true if the numbers are equal.
	return strings.HasSuffix(firstNumberNationalNumber, secondNumberNationalNumber) ||
		strings.HasSuffix(secondNumberNationalNumber, firstNumberNationalNumber)
}

// Takes two phone numbers as strings and compares them for equality. This is
// a convenience wrapper for isNumberMatch(PhoneNumber, PhoneNumber). No
// default region is known.
func isNumberMatch(firstNumber, secondNumber string) MatchType {
	// TODO(ttacon): this is bloody ridiculous, fix this function to not try things
	// and to propogate errors on through
	firstNumberAsProto, err := parse(firstNumber, UNKNOWN_REGION)
	if err == nil {
		return isNumberMatchWithOneNumber(firstNumberAsProto, secondNumber)
	}
	if err != ErrInvalidCountryCode {
		return NOT_A_NUMBER
	}

	secondNumberAsProto, err := parse(secondNumber, UNKNOWN_REGION)
	if err == nil {
		return isNumberMatchWithOneNumber(secondNumberAsProto, firstNumber)
	}
	if err != ErrInvalidCountryCode {
		return NOT_A_NUMBER
	}

	var firstNumberProto, secondNumberProto *PhoneNumber
	err = parseHelper(firstNumber, "", false, false, firstNumberProto)
	if err != nil {
		return NOT_A_NUMBER
	}
	err = parseHelper(secondNumber, "", false, false, secondNumberProto)
	if err != nil {
		return NOT_A_NUMBER
	}
	return isNumberMatchWithNumbers(firstNumberProto, secondNumberProto)
}

// Takes two phone numbers and compares them for equality. This is a
// convenience wrapper for isNumberMatch(PhoneNumber, PhoneNumber). No
// default region is known.
func isNumberMatchWithOneNumber(
	firstNumber *PhoneNumber, secondNumber string) MatchType {
	// First see if the second number has an implicit country calling
	// code, by attempting to parse it.
	secondNumberAsProto, err := parse(secondNumber, UNKNOWN_REGION)
	if err == nil {
		return isNumberMatchWithNumbers(firstNumber, secondNumberAsProto)
	}
	if err != ErrInvalidCountryCode {
		return NOT_A_NUMBER
	}
	// The second number has no country calling code. EXACT_MATCH is no
	// longer possible. We parse it as if the region was the same as that
	// for the first number, and if EXACT_MATCH is returned, we replace
	// this with NSN_MATCH.
	firstNumberRegion := getRegionCodeForCountryCode(int(firstNumber.GetCountryCode()))

	if firstNumberRegion != UNKNOWN_REGION {
		secondNumberWithFirstNumberRegion, err :=
			parse(secondNumber, firstNumberRegion)
		if err != nil {
			return NOT_A_NUMBER
		}
		match := isNumberMatchWithNumbers(
			firstNumber, secondNumberWithFirstNumberRegion)
		if match == EXACT_MATCH {
			return NSN_MATCH
		}
		return match
	} else {
		// If the first number didn't have a valid country calling
		// code, then we parse the second number without one as well.
		var secondNumberProto *PhoneNumber
		err := parseHelper(secondNumber, "", false, false, secondNumberProto)
		if err != nil {
			return NOT_A_NUMBER
		}
		return isNumberMatchWithNumbers(firstNumber, secondNumberProto)
	}

	// One or more of the phone numbers we are trying to match is not
	// a viable phone number.
	return NOT_A_NUMBER
}

// Returns true if the number can be dialled from outside the region, or
// unknown. If the number can only be dialled from within the region,
// returns false. Does not check the number is a valid number. Note that,
// at the moment, this method does not handle short numbers.
// TODO: Make this method public when we have enough metadata to make it worthwhile.
// @VisibleForTesting
func canBeInternationallyDialled(number *PhoneNumber) bool {
	metadata := getMetadataForRegion(getRegionCodeForNumber(number))
	if metadata == nil {
		// Note numbers belonging to non-geographical entities
		// (e.g. +800 numbers) are always internationally diallable,
		// and will be caught here.
		return true
	}
	nationalSignificantNumber := getNationalSignificantNumber(number)
	return !isNumberMatchingDesc(
		nationalSignificantNumber, metadata.GetNoInternationalDialling())
}

// Returns true if the supplied region supports mobile number portability.
// Returns false for invalid, unknown or regions that don't support mobile
// number portability.
func isMobileNumberPortableRegion(regionCode string) bool {
	metadata := getMetadataForRegion(regionCode)
	if metadata == nil {
		glog.Warning("Invalid or unknown region code provided: " + regionCode)
		return false
	}
	return metadata.GetMobileNumberPortableRegion()
}

func init() {
	getInstance()
	loadMetadataFromFile("", "US", 1, instance.metadataLoader)
}