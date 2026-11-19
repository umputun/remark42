package regexp2

import "github.com/dlclark/regexp2/v2/syntax"

// compileFindFirstChar preserves the precedence of the interpreter's search
// strategies: fixed trailing anchors, leading anchors, Boyer-Moore, analyzed
// searches, then the first-character fallback. Input-dependent choices such
// as the dense multi-prefix heuristic still run inside the selected finder.
// The functions read r.code so the full and capture-elided programs can share
// a finder without retaining either program in a closure.
func compileFindFirstChar(code *syntax.Code) func(*Runner) bool {
	if opts := code.FindOptimizations; opts != nil {
		switch opts.FindMode {
		case syntax.TrailingAnchor_FixedLength_LeftToRight_End:
			return findTrailingEnd
		case syntax.TrailingAnchor_FixedLength_LeftToRight_EndZ:
			return findTrailingEndZ
		}
	}
	if code.Anchors&(syntax.AnchorBeginning|syntax.AnchorStart|syntax.AnchorEndZ|syntax.AnchorEnd) != 0 {
		if code.RightToLeft {
			return findAnchoredRightToLeft
		}
		return findAnchoredLeftToRight
	}
	if code.BmPrefix != nil {
		if code.RightToLeft {
			return findBoyerMooreRightToLeft
		}
		return findBoyerMooreLeftToRight
	}
	if opts := code.FindOptimizations; opts != nil {
		switch opts.FindMode {
		case syntax.LeadingString_OrdinalIgnoreCase_LeftToRight:
			return findLeadingStringIgnoreCase
		case syntax.LeadingStrings_LeftToRight:
			return findLeadingStrings
		case syntax.LeadingStrings_OrdinalIgnoreCase_LeftToRight:
			return findLeadingStringsIgnoreCase
		case syntax.FixedDistanceChar_LeftToRight:
			return findFixedDistanceChar
		case syntax.FixedDistanceString_LeftToRight:
			return findFixedDistanceString
		case syntax.FixedDistanceSets_LeftToRight:
			return findFixedDistanceSets
		case syntax.LiteralAfterLoop_LeftToRight:
			return findLiteralAfterLoop
		case syntax.RequiredLandmarkChain_LeftToRight:
			return findLandmarkChain
		case syntax.LeadingSet_LeftToRight:
			// General Unicode sets and large enumerated sets use the class's
			// membership test in the fallback instead of a linear needle scan.
			if len(opts.FixedDistanceSets) > 0 {
				set := opts.FixedDistanceSets[0]
				if len(set.Chars) > 0 && len(set.Chars) <= 5 || set.Range != nil {
					return findFixedDistanceSets
				}
			}
		}
	}
	if code.FcPrefix == nil {
		return findWithoutPrefix
	}
	if code.FcPrefix.PrefixSet.IsSingleton() {
		if code.RightToLeft {
			return findSingletonRightToLeft
		}
		return findSingletonLeftToRight
	}
	if code.RightToLeft {
		return findSetRightToLeft
	}
	return findSetLeftToRight
}

func findTrailingEnd(r *Runner) bool {
	return findTrailingFixedLengthEnd(r, r.code.FindOptimizations.MinRequiredLength, false)
}

func findTrailingEndZ(r *Runner) bool {
	return findTrailingFixedLengthEnd(r, r.code.FindOptimizations.MinRequiredLength, true)
}

func findLeadingStringIgnoreCase(r *Runner) bool {
	opts := r.code.FindOptimizations
	return findLeadingStringLeftToRight(r, opts.LeadingPrefixRunes, true, opts.LeadingPrefixASCII)
}

func findLeadingStrings(r *Runner) bool {
	opts := r.code.FindOptimizations
	if search := r.re.prefixSearch; search != nil && search.shouldUseRunes(r.Runtext, r.Runtextpos) {
		start := search.indexRunes(r.Runtext, r.Runtextpos)
		if start >= 0 && hasRequiredLengthAt(r, start) {
			r.Runtextpos = start
			return true
		}
		r.Runtextpos = r.Runtextend
		return false
	}
	return findLeadingStringsLeftToRight(r, opts.LeadingPrefixesRunes, opts.LeadingPrefixFirstRunes, opts.LeadingPrefixFirstASCII, false)
}

func findLeadingStringsIgnoreCase(r *Runner) bool {
	opts := r.code.FindOptimizations
	return findLeadingStringsLeftToRight(r, opts.LeadingPrefixesRunes, opts.LeadingPrefixFirstRunes, opts.LeadingPrefixFirstASCII, true)
}

func findFixedDistanceChar(r *Runner) bool {
	literal := r.code.FindOptimizations.FixedDistanceLiteral
	return findFixedDistanceCharLeftToRight(r, literal.C, literal.Distance)
}

func findFixedDistanceString(r *Runner) bool {
	literal := r.code.FindOptimizations.FixedDistanceLiteral
	return findFixedDistanceStringLeftToRight(r, literal.Runes, literal.Distance)
}

func findFixedDistanceSets(r *Runner) bool {
	return findFixedDistanceSetsLeftToRight(r, r.code.FindOptimizations.FixedDistanceSets)
}

func findLiteralAfterLoop(r *Runner) bool {
	return findLiteralAfterLoopLeftToRight(r, r.code.FindOptimizations.LiteralAfterLoop)
}

func findLandmarkChain(r *Runner) bool {
	return findRequiredLandmarkChainLeftToRight(r, r.code.FindOptimizations.LandmarkChain)
}

func findAnchoredLeftToRight(r *Runner) bool {
	anchors := r.code.Anchors
	if anchors&syntax.AnchorBeginning != 0 && r.Runtextpos > 0 ||
		anchors&syntax.AnchorStart != 0 && r.Runtextpos > r.Runtextstart {
		r.Runtextpos = r.Runtextend
		return false
	}
	if anchors&syntax.AnchorEndZ != 0 && r.Runtextpos < r.Runtextend-1 {
		r.Runtextpos = r.Runtextend - 1
	} else if anchors&syntax.AnchorEnd != 0 && r.Runtextpos < r.Runtextend {
		r.Runtextpos = r.Runtextend
	}
	return r.code.BmPrefix == nil || r.code.BmPrefix.IsMatch(r.Runtext, r.Runtextpos, 0, r.Runtextend)
}

func findAnchoredRightToLeft(r *Runner) bool {
	anchors := r.code.Anchors
	if anchors&syntax.AnchorEnd != 0 && r.Runtextpos < r.Runtextend ||
		anchors&syntax.AnchorEndZ != 0 && (r.Runtextpos < r.Runtextend-1 ||
			(r.Runtextpos == r.Runtextend-1 && r.Runtext[r.Runtextpos] != '\n')) ||
		anchors&syntax.AnchorStart != 0 && r.Runtextpos < r.Runtextstart {
		r.Runtextpos = 0
		return false
	}
	if anchors&syntax.AnchorBeginning != 0 && r.Runtextpos > 0 {
		r.Runtextpos = 0
	}
	return r.code.BmPrefix == nil || r.code.BmPrefix.IsMatch(r.Runtext, r.Runtextpos, 0, r.Runtextend)
}

func findBoyerMooreLeftToRight(r *Runner) bool {
	r.Runtextpos = r.code.BmPrefix.Scan(r.Runtext, r.Runtextpos, 0, r.Runtextend)
	if r.Runtextpos < 0 {
		r.Runtextpos = r.Runtextend
		return false
	}
	return true
}

func findBoyerMooreRightToLeft(r *Runner) bool {
	r.Runtextpos = r.code.BmPrefix.Scan(r.Runtext, r.Runtextpos, 0, r.Runtextend)
	if r.Runtextpos < 0 {
		r.Runtextpos = 0
		return false
	}
	return true
}

func findWithoutPrefix(*Runner) bool { return true }

func findSingletonLeftToRight(r *Runner) bool {
	ch := r.code.FcPrefix.PrefixSet.SingletonChar()
	for r.Runtextpos < r.Runtextend {
		if r.Runtext[r.Runtextpos] == ch {
			return true
		}
		r.Runtextpos++
	}
	return false
}

func findSingletonRightToLeft(r *Runner) bool {
	ch := r.code.FcPrefix.PrefixSet.SingletonChar()
	for r.Runtextpos > 0 {
		if r.Runtext[r.Runtextpos-1] == ch {
			return true
		}
		r.Runtextpos--
	}
	return false
}

func findSetLeftToRight(r *Runner) bool {
	set := r.code.FcPrefix.PrefixSet
	for r.Runtextpos < r.Runtextend {
		if set.Contains(r.Runtext[r.Runtextpos]) {
			return true
		}
		r.Runtextpos++
	}
	return false
}

func findSetRightToLeft(r *Runner) bool {
	set := r.code.FcPrefix.PrefixSet
	for r.Runtextpos > 0 {
		if set.Contains(r.Runtext[r.Runtextpos-1]) {
			return true
		}
		r.Runtextpos--
	}
	return false
}
