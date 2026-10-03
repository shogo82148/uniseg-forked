package uniseg

import "unicode/utf8"

// The states of the line break parser. The lower bits of a state contain the
// line breaking class of the last character (after the resolution of rules LB1,
// LB9, and LB10). Some classes are split into multiple states to capture
// sequences relevant to the rules (e.g. spaces following certain classes).
// The higher bits contain additional flags and context information.
const (
	lbAny = iota // Start of text (sot).
	lbBK
	lbCR
	lbLF
	lbNL
	lbSP
	lbZW
	lbZWSP   // ZW SP+ (LB8)
	lbOPSP   // OP SP+ (LB14)
	lbQUPiSP // [\p{Pi}&QU] SP+ (LB15a)
	lbCLCPSP // (CL | CP) SP+ (LB16)
	lbB2SP   // B2 SP+ (LB17)
	lbWJ
	lbGL
	lbBA
	lbHY
	lbHH
	lbCL
	lbCP
	lbEX
	lbIS
	lbSY
	lbOP
	lbQU   // QU which is neither Pi nor Pf.
	lbQUPi // [\p{Pi}&QU]
	lbQUPf // [\p{Pf}&QU]
	lbNS
	lbB2
	lbCB
	lbBB
	lbHL
	lbAL
	lbNU
	lbPR
	lbPO
	lbID
	lbEB
	lbEM
	lbIN
	lbJL
	lbJV
	lbJT
	lbH2
	lbH3
	lbRI
	lbAK
	lbAP
	lbAS
	lbVI
	lbVF

	lbClassMask = 63 // Extracts the line breaking class from a state.

	lbZWJBit = 64  // The last character was a zero width joiner (LB8a).
	lbEABit  = 128 // The last base character is East Asian, i.e. it has East_Asian_Width F, W, or H (LB19a, LB30).

	// Context bits. Their meaning depends on the line breaking class.
	//
	//   - lbQU, lbQUPi, lbQUPf: lbCtxA = the quotation mark follows sot or a
	//     non-East Asian character (LB19a). lbCtxB = the initial quotation
	//     mark follows sot, BK, CR, LF, NL, OP, QU, GL, SP, or ZW (LB15a).
	//   - lbHY, lbHH: lbCtxA = the hyphen follows sot, BK, CR, LF, NL, SP, ZW,
	//     CB, or GL (LB20a). lbCtxB = the hyphen follows HL (LB21a).
	//   - lbSY, lbIS, lbCL, lbCP: lbCtxA = the character is part of the
	//     sequence NU (SY | IS)* (CL | CP)? (LB25).
	//   - lbAL: lbCtxA = U+25CC DOTTED CIRCLE (LB28a).
	//   - lbAL, lbID: lbCtxB = the character is Extended_Pictographic and
	//     unassigned (LB30b).
	//   - lbVI: lbCtxA = the virama follows AK, U+25CC, or AS (LB28a).
	//   - lbRI: lbCtxA = there is an odd number of regional indicators (LB30a).
	lbCtxA = 256
	lbCtxB = 512
)

// These constants define whether a given text may be broken into the next line.
// If the break is optional (LineCanBreak), you may choose to break or not based
// on your own criteria, for example, if the text has reached the available
// width.
const (
	LineDontBreak = iota // You may not break the line here.
	LineCanBreak         // You may or may not break the line here.
	LineMustBreak        // You must break the line here.
)

// lbClasses maps line break properties (after resolution according to LB1) to
// line break classes as used in the parser states. Properties which result in
// more than one possible class (QU and SP) are mapped to their base class and
// refined later.
var lbClasses = [...]int{
	prBK: lbBK,
	prCR: lbCR,
	prLF: lbLF,
	prNL: lbNL,
	prSP: lbSP,
	prZW: lbZW,
	prWJ: lbWJ,
	prGL: lbGL,
	prBA: lbBA,
	prHY: lbHY,
	prHH: lbHH,
	prCL: lbCL,
	prCP: lbCP,
	prEX: lbEX,
	prIS: lbIS,
	prSY: lbSY,
	prOP: lbOP,
	prQU: lbQU,
	prNS: lbNS,
	prB2: lbB2,
	prCB: lbCB,
	prBB: lbBB,
	prHL: lbHL,
	prAL: lbAL,
	prNU: lbNU,
	prPR: lbPR,
	prPO: lbPO,
	prID: lbID,
	prEB: lbEB,
	prEM: lbEM,
	prIN: lbIN,
	prJL: lbJL,
	prJV: lbJV,
	prJT: lbJT,
	prH2: lbH2,
	prH3: lbH3,
	prRI: lbRI,
	prAK: lbAK,
	prAP: lbAP,
	prAS: lbAS,
	prVI: lbVI,
	prVF: lbVF,
}

// dottedCircle is U+25CC DOTTED CIRCLE which is used in LB28a.
const dottedCircle = 0x25cc

// lineBreakClass returns the line breaking class of the given code point after
// applying rule LB1. It returns -1 for combining marks (CM) and zero width
// joiners (ZWJ). The second return value is the code point's General Category.
func lineBreakClass(r rune) (class, generalCategory int) {
	var prop int
	prop, generalCategory = propertyWithGenCat(lineBreakCodePoints, r)

	// LB1.
	switch prop {
	case prAI, prSG, prXX:
		prop = prAL
	case prSA:
		if generalCategory == gcMn || generalCategory == gcMc {
			prop = prCM
		} else {
			prop = prAL
		}
	case prCJ:
		prop = prNS
	}

	if prop == prCM || prop == prZWJ {
		return -1, generalCategory
	}
	if prop < len(lbClasses) {
		if class = lbClasses[prop]; class != lbAny {
			return class, generalCategory
		}
	}
	return lbAL, generalCategory // Should not happen.
}

// isEastAsian returns true if the given code point has East_Asian_Width F, W,
// or H.
func isEastAsian(r rune) bool {
	if r < 0x1100 {
		return false // Fast path: There are no such characters below U+1100.
	}
	ea := property(eastAsianWidth, r)
	return ea == prF || ea == prW || ea == prH
}

// lbLookahead holds information about the characters following the character
// currently being processed by the line break parser.
type lbLookahead struct {
	b   []byte
	str string
}

// next returns the next code point which is not a combining mark or a zero
// width joiner (these are ignored according to LB9) and its line breaking class.
// If the end of the text is reached, ok is false.
func (l *lbLookahead) next() (r rune, class int, ok bool) {
	for {
		var length int
		if l.b != nil { // Byte slice version.
			if len(l.b) == 0 {
				return 0, 0, false
			}
			r, length = utf8.DecodeRune(l.b)
			l.b = l.b[length:]
		} else { // String version.
			if len(l.str) == 0 {
				return 0, 0, false
			}
			r, length = utf8.DecodeRuneInString(l.str)
			l.str = l.str[length:]
		}
		class, _ = lineBreakClass(r)
		if class >= 0 {
			return r, class, true
		}
	}
}

// transitionLineBreakState determines the new state of the line break parser
// given the current state and the next code point. It also returns the type of
// line break: LineDontBreak, LineCanBreak, or LineMustBreak. If more than one
// code point is needed to determine the new state, the byte slice or the string
// starting after rune "r" can be used (whichever is not nil or empty) for
// further lookups.
//
// The rules are those of Unicode Standard Annex #14, Unicode version 18.0.0.
func transitionLineBreakState(state int, r rune, b []byte, str string) (newState int, lineBreak int) {
	if state < 0 {
		state = lbAny // LB2: sot is treated like a state without previous characters.
	}
	prev := state & lbClassMask
	prevEA := state&lbEABit != 0
	ctxA := state&lbCtxA != 0
	ctxB := state&lbCtxB != 0

	// Determine the class of the next character.
	next, generalCategory := lineBreakClass(r)

	// Combining marks.
	if next < 0 {
		var zwj int
		if r == 0x200d {
			zwj = lbZWJBit
		}
		switch prev {
		case lbAny, lbBK, lbCR, lbLF, lbNL, lbSP, lbZW, lbZWSP, lbOPSP, lbQUPiSP, lbCLCPSP, lbB2SP:
			// LB10: Treat any remaining CM or ZWJ as AL with the properties of
			// U+0041 LATIN CAPITAL LETTER A.
			next = lbAL
			generalCategory = gcLu
			r = 'A'
			defer func() {
				newState |= zwj
			}()
		default:
			// LB9: Treat X (CM | ZWJ)* as X.
			return (state &^ lbZWJBit) | zwj, LineDontBreak
		}
	}

	nextEA := isEastAsian(r)
	lookahead := lbLookahead{b: b, str: str}

	// Determine the new state.
	newState = next
	switch next {
	case lbSP:
		switch prev {
		case lbZW, lbZWSP:
			newState = lbZWSP
		case lbOP, lbOPSP:
			newState = lbOPSP
		case lbQUPi:
			if ctxB {
				newState = lbQUPiSP
			}
		case lbQUPiSP:
			newState = lbQUPiSP
		case lbCL, lbCP, lbCLCPSP:
			newState = lbCLCPSP
		case lbB2, lbB2SP:
			newState = lbB2SP
		}
	case lbQU:
		switch generalCategory {
		case gcPi:
			newState = lbQUPi
			switch prev {
			case lbAny, lbBK, lbCR, lbLF, lbNL, lbOP, lbQU, lbQUPi, lbQUPf, lbGL, lbSP, lbZW, lbZWSP, lbOPSP, lbQUPiSP, lbCLCPSP, lbB2SP:
				newState |= lbCtxB
			}
		case gcPf:
			newState = lbQUPf
		}
		next = newState
		if prev == lbAny || !prevEA {
			newState |= lbCtxA
		}
	case lbHY, lbHH:
		switch prev {
		case lbAny, lbBK, lbCR, lbLF, lbNL, lbSP, lbZW, lbZWSP, lbOPSP, lbQUPiSP, lbCLCPSP, lbB2SP, lbCB, lbGL:
			newState |= lbCtxA
		case lbHL:
			newState |= lbCtxB
		}
	case lbSY, lbIS, lbCL, lbCP:
		if prev == lbNU || (prev == lbSY || prev == lbIS) && ctxA {
			newState |= lbCtxA
		}
	case lbAL:
		if r == dottedCircle {
			newState |= lbCtxA
		}
		fallthrough
	case lbID:
		// Unassigned code points are not listed in the property table and
		// have no General Category there.
		if (generalCategory == gcCn || generalCategory == gcNone) && property(graphemeCodePoints, r) == prExtendedPictographic {
			newState |= lbCtxB
		}
	case lbVI:
		if prev == lbAK || prev == lbAS || prev == lbAL && ctxA {
			newState |= lbCtxA
		}
	case lbRI:
		if prev != lbRI || !ctxA {
			newState |= lbCtxA
		}
	}
	if nextEA {
		newState |= lbEABit
	}

	// Determine the line break.
	lineBreak = lineBreakBetween(prev, next, newState, prevEA, nextEA, ctxA, ctxB, state&lbZWJBit != 0, r, &lookahead)
	return
}

// lineBreakBetween applies the line breaking rules LB4 to LB31 to determine the
// line break between a character of class "prev" and one of class "next" (code
// point "r"). "newState" is the parser state after the "next" character. The
// other arguments describe properties of the "prev" and "next" characters, see
// [transitionLineBreakState] for details.
func lineBreakBetween(prev, next, newState int, prevEA, nextEA, ctxA, ctxB, zwj bool, r rune, lookahead *lbLookahead) int {
	isSP := func(class int) bool {
		switch class {
		case lbSP, lbZWSP, lbOPSP, lbQUPiSP, lbCLCPSP, lbB2SP:
			return true
		}
		return false
	}
	isAKLike := func(class int, dottedCircle bool) bool { // (AK | [◌] | AS)
		return class == lbAK || class == lbAS || class == lbAL && dottedCircle
	}
	nextIsDottedCircle := next == lbAL && newState&lbCtxA != 0
	isNUSeq := prev == lbNU || (prev == lbSY || prev == lbIS) && ctxA // NU (SY | IS)*

	// LB4.
	if prev == lbBK {
		return LineMustBreak
	}

	// LB5.
	if prev == lbCR && next == lbLF {
		return LineDontBreak
	}
	if prev == lbCR || prev == lbLF || prev == lbNL {
		return LineMustBreak
	}

	// LB6.
	if next == lbBK || next == lbCR || next == lbLF || next == lbNL {
		return LineDontBreak
	}

	// LB7.
	if next == lbSP || next == lbZW {
		return LineDontBreak
	}

	// LB8.
	if prev == lbZW || prev == lbZWSP {
		return LineCanBreak
	}

	// LB8a.
	if zwj {
		return LineDontBreak
	}

	// LB11.
	if next == lbWJ || prev == lbWJ {
		return LineDontBreak
	}

	// LB12.
	if prev == lbGL {
		return LineDontBreak
	}

	// LB12a.
	if next == lbGL && !isSP(prev) && prev != lbHY && prev != lbHH {
		return LineDontBreak
	}

	// LB13.
	if next == lbCL || next == lbCP || next == lbEX || next == lbSY {
		return LineDontBreak
	}

	// LB14.
	if prev == lbOP || prev == lbOPSP {
		return LineDontBreak
	}

	// LB15a.
	if prev == lbQUPi && ctxB || prev == lbQUPiSP {
		return LineDontBreak
	}

	// LB15b.
	if next == lbQUPf {
		l := *lookahead
		_, class, ok := l.next()
		if !ok {
			return LineDontBreak // eot
		}
		switch class {
		case lbSP, lbGL, lbWJ, lbCL, lbQU, lbCP, lbEX, lbIS, lbSY, lbBK, lbCR, lbLF, lbNL, lbZW:
			return LineDontBreak
		}
	}

	// LB15c.
	if isSP(prev) && next == lbIS {
		l := *lookahead
		if _, class, ok := l.next(); ok && class == lbNU {
			return LineCanBreak
		}
	}

	// LB15d.
	if next == lbIS {
		return LineDontBreak
	}

	// LB16.
	if (prev == lbCL || prev == lbCP || prev == lbCLCPSP) && next == lbNS {
		return LineDontBreak
	}

	// LB17.
	if (prev == lbB2 || prev == lbB2SP) && next == lbB2 {
		return LineDontBreak
	}

	// LB18.
	if isSP(prev) {
		return LineCanBreak
	}

	prevIsQU := prev == lbQU || prev == lbQUPi || prev == lbQUPf
	nextIsQU := next == lbQU || next == lbQUPi || next == lbQUPf

	// LB19.
	if next == lbQU || next == lbQUPf || prev == lbQU || prev == lbQUPi {
		return LineDontBreak
	}

	// LB19a.
	if nextIsQU {
		if !prevEA {
			return LineDontBreak
		}
		l := *lookahead
		if r, _, ok := l.next(); !ok || !isEastAsian(r) {
			return LineDontBreak
		}
	}
	if prevIsQU && (!nextEA || ctxA) {
		return LineDontBreak
	}

	// LB20.
	if next == lbCB || prev == lbCB {
		return LineCanBreak
	}

	// LB20a.
	if (prev == lbHY || prev == lbHH) && ctxA && (next == lbAL || next == lbHL) {
		return LineDontBreak
	}

	// LB21.
	if next == lbBA || next == lbHH || next == lbHY || next == lbNS || prev == lbBB {
		return LineDontBreak
	}

	// LB21a.
	if (prev == lbHY || prev == lbHH) && ctxB && next != lbHL {
		return LineDontBreak
	}

	// LB21b.
	if prev == lbSY && next == lbHL {
		return LineDontBreak
	}

	// LB22.
	if next == lbIN {
		return LineDontBreak
	}

	// LB23.
	if (prev == lbAL || prev == lbHL) && next == lbNU {
		return LineDontBreak
	}
	if prev == lbNU && (next == lbAL || next == lbHL) {
		return LineDontBreak
	}

	// LB23a.
	if prev == lbPR && (next == lbID || next == lbEB || next == lbEM) {
		return LineDontBreak
	}
	if (prev == lbID || prev == lbEB || prev == lbEM) && next == lbPO {
		return LineDontBreak
	}

	// LB24.
	if (prev == lbPR || prev == lbPO) && (next == lbAL || next == lbHL) {
		return LineDontBreak
	}
	if (prev == lbAL || prev == lbHL) && (next == lbPR || next == lbPO) {
		return LineDontBreak
	}

	// LB25.
	if (next == lbPO || next == lbPR) && (isNUSeq || (prev == lbCL || prev == lbCP) && ctxA) {
		return LineDontBreak
	}
	if (prev == lbPO || prev == lbPR) && next == lbOP {
		l := *lookahead
		_, class, ok := l.next()
		if ok && class == lbIS {
			_, class, ok = l.next()
		}
		if ok && class == lbNU {
			return LineDontBreak
		}
	}
	if next == lbNU && (prev == lbPO || prev == lbPR || prev == lbHY || prev == lbIS || isNUSeq) {
		return LineDontBreak
	}

	// LB26.
	if prev == lbJL && (next == lbJL || next == lbJV || next == lbH2 || next == lbH3) {
		return LineDontBreak
	}
	if (prev == lbJV || prev == lbH2) && (next == lbJV || next == lbJT) {
		return LineDontBreak
	}
	if (prev == lbJT || prev == lbH3) && next == lbJT {
		return LineDontBreak
	}

	// LB27.
	switch prev {
	case lbJL, lbJV, lbJT, lbH2, lbH3:
		if next == lbPO {
			return LineDontBreak
		}
	case lbPR:
		switch next {
		case lbJL, lbJV, lbJT, lbH2, lbH3:
			return LineDontBreak
		}
	}

	// LB28.
	if (prev == lbAL || prev == lbHL) && (next == lbAL || next == lbHL) {
		return LineDontBreak
	}

	// LB28a.
	nextIsAKLike := isAKLike(next, nextIsDottedCircle)
	prevIsAKLike := isAKLike(prev, ctxA)
	if prev == lbAP && nextIsAKLike {
		return LineDontBreak
	}
	if prevIsAKLike && (next == lbVF || next == lbVI) {
		return LineDontBreak
	}
	if prev == lbVI && ctxA && (next == lbAK || nextIsDottedCircle) {
		return LineDontBreak
	}
	if prevIsAKLike && nextIsAKLike {
		l := *lookahead
		if _, class, ok := l.next(); ok && class == lbVF {
			return LineDontBreak
		}
	}

	// LB29.
	if prev == lbIS && (next == lbAL || next == lbHL) {
		return LineDontBreak
	}

	// LB30.
	if (prev == lbAL || prev == lbHL || prev == lbNU) && next == lbOP && !nextEA {
		return LineDontBreak
	}
	if prev == lbCP && !prevEA && (next == lbAL || next == lbHL || next == lbNU) {
		return LineDontBreak
	}

	// LB30a.
	if prev == lbRI && ctxA && next == lbRI {
		return LineDontBreak
	}

	// LB30b.
	if next == lbEM && (prev == lbEB || (prev == lbAL || prev == lbID) && ctxB) {
		return LineDontBreak
	}

	// LB31.
	return LineCanBreak
}
