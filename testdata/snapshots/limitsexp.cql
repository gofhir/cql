library Differential version '1.0'
// Written by hand: the Exp and Ln of docs/plans/013-null-donde-va-null.md, apart
// from the steps of limits.cql because they are a float's and compared past its
// last digits. CQL 1.5.3, Appendix B, Exp and Ln: "If the result of the
// operation cannot be represented, the result is null".
define D86: Exp(1000)
define D87: Exp(1000.0)
define D88: Exp(709)
define D89: Exp(100)
define D90: Exp(64)
define D91: Exp(50)
define D92: Exp(46.06)
define D93: Exp(46.05)
define D94: Exp(46)
define D95: Exp(-1000)
define D96: Exp(-46.05)
define D97: Exp(0)
define D98: Exp(1000) is null
define D99: Coalesce(Exp(1000), 1.0)
define D100: Ln(0)
define D101: Ln(0.0)
define D102: Ln(-0)
define D103: Ln(-0.0)
define D104: Ln(-1)
define D105: Ln(1)
define D106: Ln(0.00000001)
define D107: Coalesce(Ln(0), 7)
define D108: Ln(0) + 1
