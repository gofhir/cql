#!/usr/bin/env python3
# Writes testdata/snapshots/fractionaloffset.cql, the area of plan 024
# (docs/plans/024-same-precision-offset.md), to standard output; with --oracle
# the answers of its independent oracle; with --models what each reading of an
# offset would answer, to classify the divergences with the references.
# Usage: python3 scripts/fractionaloffset-area.py > testdata/snapshots/fractionaloffset.cql
#        python3 scripts/fractionaloffset-area.py --oracle > testdata/fractionaloffset/oracle.tsv
#        python3 scripts/fractionaloffset-area.py --models > models.tsv
#
# The area is evaluated at +05:30 (internal/referenceharness/evaloffset), where
# UTC and the evaluation offset cut the hours at different instants. CQL 1.5.3,
# Appendix B, Same As (https://cql.hl7.org/N1A/09-b-cqlreference.html): "When
# comparing DateTime values with different timezone offsets, implementations
# should normalize to the timezone offset of the evaluation request timestamp,
# but only when the comparison precision is hours, minutes, seconds, or
# milliseconds." Same Or Before, Same Or After, Before and After say the same.
#
# First the probes of the plan, D1 to D21 as it numbers them, then the
# expressions of the attack of its adversarial round; then an enumeration of
# every operator with a precision, at hour, minute, second and millisecond,
# over pairs of values with no offset, with one offset on both sides, with
# different offsets, and across a day boundary.
#
# After those, appended so that the names of the rows above stay as they were, the
# enumeration of plan 025 (docs/plans/025-hour-spread-by-offset.md): a DateTime to
# the hour that the offset of the evaluation cuts in two, `@2020-01-01T00Z` being
# 05:30 to 06:29 at +05:30, is its span of instants and not the hour 5 or the 6.
# Hour values with offsets Z, +01:00, +05:30, +05:45, -03:00 and -09:30, against
# values to the hour, the minute and the second at positions before, at the edges
# of, inside and after the span, with the five point operators, `in`, `includes`
# and `meets`, at the hour and the minute, across a change of day.
#
# The oracle is pure Python and shares nothing with the engine. A DateTime that
# writes no offset has the evaluation's. Two with one offset are compared in the
# digits written; two with different offsets are both moved to the offset of the
# evaluation; each is then cut at the precision and compared.
import re
import sys
from datetime import datetime, timedelta

EVAL = 5 * 60 + 30  # minutes: the evaluation offset of this area, +05:30
# cql-execution truncates a fractional evaluation offset to whole hours when it
# places a value with no offset (lib/elm/literal.js parseInt): +05:30 is +05:00.
JS_EVAL = 5 * 60

PROBES = """@2020-01-01T00:10:00Z same hour as @2020-01-01T01:40:00+01:00
@2020-01-01T00:10:00Z same hour as @2020-01-01T00:40:00Z
hours between @2020-01-01T00:10:00Z and @2020-01-01T00:40:00Z
difference in hours between @2020-01-01T00:10:00Z and @2020-01-01T00:40:00Z
@2020-01-01T00:10:00Z same hour as @2020-01-01T00:40:00
@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00
@2020-01-01T18:20:00Z same day as @2020-01-01T18:40:00Z
@2020-01-01T18:20:00Z same hour as @2020-01-01T18:40:00Z
hour from @2020-01-01T00:10:00Z
@2020-01-01T00:10:00Z = @2020-01-01T05:40:00+05:30
@2020-01-01T00:10:00.000Z same hour as @2020-01-01T00:40:00.000Z
@2020-01-01T00:10:00Z same hour or before @2020-01-01T00:40:00Z
@2020-01-01T00:40:00Z same hour or after @2020-01-01T00:10:00Z
@2020-01-01T00:10:00Z same minute as @2020-01-01T00:10:30Z
@2020-01-01T00:10:00 same day as @2020-01-01T23:50:00
@2020-01-01T00:10:00Z same day as @2020-01-01T23:50:00Z
@2020-01-01T00:10:00Z in hour of Interval[@2020-01-01T00:40:00Z, @2020-01-01T02:00:00Z]
Interval[@2020-01-01T00:10:00Z, @2020-01-01T00:20:00Z] during hour of Interval[@2020-01-01T00:40:00Z, @2020-01-01T02:00:00Z]
@2020-01-01T00:10:00Z before hour of @2020-01-01T01:20:00Z
@2020-01-01T00:10:00 before hour of @2020-01-01T00:40:00
@2020-01-01T00:10:00 same hour as @2020-01-01T00:40:00+05:30""".split("\n")

# The attack of the adversarial round, without the expand and the collapse
# (D5 to D7 of it), which do not depend on the evaluation offset (plan 024,
# round 1, M2).
ATTACK = """@2020-01-01T00Z = @2020-01-01T01:40+01:00
@2020-01-01T00Z < @2020-01-01T01:40+01:00
@2020-01-01T00:10:00+01:00 same hour as @2020-01-01T00:40:00+01:00
difference in hours between @2020-01-01T00:10:00+01:00 and @2020-01-01T00:40:00+01:00
Interval[@2020-01-01T00:10:00, @2020-01-01T00:20:00] meets hour of Interval[@2020-01-01T00:40:00, @2020-01-01T01:00:00]
Interval[@2020-01-01T00:10:00, @2020-01-01T00:20:00] same as Interval[@2020-01-01T00Z, @2020-01-01T01:40+01:00]
@2020-01-01T05Z = @2020-01-01T06:10Z
@2020-01-01T05Z < @2020-01-01T06:10Z
Interval[@2019-12-31T23:00Z, @2020-01-01T00Z] same or before Interval[@2020-01-01T01:40+01:00, @2020-01-01T03:00+01:00]
@2020-01-01T00:10:00Z same hour as @2020-01-01T01:40:00+01:00
@2020-01-01T00:10:00Z same hour as @2020-01-01T00:40:00Z
hours between @2020-01-01T00:10:00Z and @2020-01-01T00:40:00Z
difference in hours between @2020-01-01T00:10:00Z and @2020-01-01T00:40:00Z
timezoneoffset from Now()
timezoneoffset from @2020-01-01T00:10:00
@2020-01-01T00:10:00 same minute as @2020-01-01T00:10:00+05:30
@2020-01-01T00Z same as @2020-01-01T01:40+01:00
@2020-01-01T05+05:30 = @2020-01-01T06:10+05:30
@2020-01-01T05+05:30 < @2020-01-01T06:10+05:30""".split("\n")

UNITS = ["hour", "minute", "second", "millisecond"]
UNIT_INDEX = {"hour": 3, "minute": 4, "second": 5, "millisecond": 6}

# Pairs of instants at each precision, each with a distance to the second: the
# values are these instants written in the offset of each side, so that the
# same pair is read in a different hour or on a different day by the frame it is
# read in. At the hour: both in one hour at Z and in two at +05:30 and the other
# way round, and across the day at Z and at +05:30. At the finer precisions the
# boundaries are the same in every offset of whole minutes, and the pairs are
# inside a unit, across it, and across the day at Z and at +05:30.
D = datetime
PAIRS = {
    "hour": [("one hour at +05:30", D(2020, 1, 1, 0, 40, 30, 250000), timedelta(minutes=30)),
             ("one hour at Z", D(2020, 1, 1, 0, 10, 30, 250000), timedelta(minutes=30)),
             ("day at Z", D(2019, 12, 31, 23, 40, 30, 250000), timedelta(minutes=30)),
             ("day at +05:30", D(2019, 12, 31, 18, 10, 30, 250000), timedelta(minutes=30))],
    "minute": [("across", D(2020, 1, 1, 0, 0, 50, 250000), timedelta(seconds=20)),
               ("day at Z", D(2019, 12, 31, 23, 59, 50, 250000), timedelta(seconds=20)),
               ("day at +05:30", D(2019, 12, 31, 18, 29, 50, 250000), timedelta(seconds=20))],
    "second": [("across", D(2020, 1, 1, 0, 0, 0, 900000), timedelta(milliseconds=200)),
               ("day at Z", D(2019, 12, 31, 23, 59, 59, 900000), timedelta(milliseconds=200)),
               ("day at +05:30", D(2019, 12, 31, 18, 29, 59, 900000), timedelta(milliseconds=200))],
    "millisecond": [("across", D(2020, 1, 1, 0, 0, 0, 250000), timedelta(milliseconds=1)),
                    ("day at Z", D(2019, 12, 31, 23, 59, 59, 999000), timedelta(milliseconds=1)),
                    ("day at +05:30", D(2019, 12, 31, 18, 29, 59, 999000), timedelta(milliseconds=1))],
}

# The offsets each side writes: None is none.
SAME = [(None, None), ("Z", "Z"), ("+05:30", "+05:30"), ("+01:00", "+01:00")]
DIFFERENT = [(None, "Z"), ("Z", None), ("+05:30", None), ("Z", "+05:30"), ("+05:30", "-03:00")]
OFFSETS = SAME + DIFFERENT

POINT_OPS = [("same {u} as", "same"), ("same {u} or before", "sob"), ("same {u} or after", "soa"),
             ("before {u} of", "before"), ("after {u} of", "after")]
INTERVAL_OPS = ["meets {u} of", "overlaps {u} of", "starts {u} of", "ends {u} of",
                "during {u} of", "includes {u} of"]


def minutes(off):
    if off is None:
        return None
    if off == "Z":
        return 0
    sign = -1 if off[0] == "-" else 1
    return sign * (int(off[1:3]) * 60 + int(off[4:6]))


def wall(instant, off, eval_minutes=EVAL):
    """An instant as the wall clock of an offset, a value with none being at the
    offset of the evaluation: what the literal writes."""
    m = eval_minutes if off is None else minutes(off)
    return instant + timedelta(minutes=m)


def lit(t, off):
    return "@" + t.strftime("%Y-%m-%dT%H:%M:%S.") + "%03d" % (t.microsecond // 1000) + (off or "")


class Frames:
    """How a pair of DateTimes is read: the offset of the evaluation, and what is
    done with two offsets that differ."""

    def __init__(self, name, eval_minutes, mode):
        self.name, self.eval, self.mode = name, eval_minutes, mode


# The reading of the plan, and those the references and the older engine have.
MODELS = [Frames("rule", EVAL, "frame"),
          Frames("left", EVAL, "left"), Frames("right", EVAL, "right"),
          Frames("utc", EVAL, "utc"), Frames("oldutc", EVAL, "alwaysutc"),
          Frames("js-left", JS_EVAL, "left"), Frames("js-right", JS_EVAL, "right"),
          Frames("js-rule", JS_EVAL, "frame")]


def cut(t, p):
    return (t.year, t.month, t.day, t.hour, t.minute, t.second, t.microsecond // 1000)[:UNIT_INDEX[p] + 1]


def read(a, b, model):
    """The two walls of a pair, in the one frame they are read in. a and b are
    (wall, offset) pairs."""
    ea = model.eval if a[1] is None else minutes(a[1])
    eb = model.eval if b[1] is None else minutes(b[1])
    if ea == eb and model.mode != "alwaysutc":
        return a[0], b[0]
    target = {"frame": model.eval, "left": ea, "right": eb, "utc": 0, "alwaysutc": 0}[model.mode]
    return (a[0] + timedelta(minutes=target - ea), b[0] + timedelta(minutes=target - eb))


def compare(a, b, p, model):
    x, y = read(a, b, model)
    cx, cy = cut(x, p), cut(y, p)
    return (cx > cy) - (cx < cy)


def successor(a, b, p, model):
    """Whether b's unit is the one after a's."""
    x, y = read(a, b, model)
    step = {"hour": timedelta(hours=1), "minute": timedelta(minutes=1), "second": timedelta(seconds=1),
            "millisecond": timedelta(milliseconds=1)}[p]
    nx = cut(x + step, p)
    # x + step may cross the boundary of the next unit only by landing in it,
    # which is what the successor is
    return nx == cut(y, p)


def point(op, a, b, p, m):
    c = compare(a, b, p, m)
    return {"same": c == 0, "sob": c <= 0, "soa": c >= 0, "before": c < 0, "after": c > 0}[op]


def interval(op, A, B, p, m):
    (sa, ea), (sb, eb) = A, B
    le = lambda x, y: compare(x, y, p, m) <= 0
    ge = lambda x, y: compare(x, y, p, m) >= 0
    eq = lambda x, y: compare(x, y, p, m) == 0
    if op == "meets":
        return successor(ea, sb, p, m) or successor(eb, sa, p, m)
    if op == "overlaps":
        return le(sa, eb) and ge(ea, sb)
    if op == "starts":
        return eq(sa, sb) and le(ea, eb)
    if op == "ends":
        return eq(ea, eb) and ge(sa, sb)
    if op == "during":
        return ge(sa, sb) and le(ea, eb)
    if op == "includes":
        return le(sa, sb) and ge(ea, eb)
    raise ValueError(op)


def build():
    """The generated rows: (expression, {model name: answer} or None)."""
    rows = [(e, None) for e in PROBES] + [(e, None) for e in ATTACK]
    for p in UNITS:
        for _, t0, delta in PAIRS[p]:
            t1 = t0 + delta
            t2 = t1 + delta
            for oa, ob in OFFSETS:
                a, b, c = (wall(t0, oa), oa), (wall(t1, ob), ob), (wall(t2, ob), ob)
                for phrase, op in POINT_OPS:
                    expr = f"{lit(*a)} {phrase.format(u=p)} {lit(*b)}"
                    rows.append((expr, {m.name: point(op, a, b, p, m) for m in MODELS}))
                # A = [a, b] at one offset and B = [b, c] at the other.
                A = (a, (wall(t1, oa), oa))
                for op in INTERVAL_OPS:
                    expr = (f"Interval[{lit(*A[0])}, {lit(*A[1])}] {op.format(u=p)} "
                            f"Interval[{lit(*b)}, {lit(*c)}]")
                    rows.append((expr, {m.name: interval(op.split()[0], A, (b, c), p, m) for m in MODELS}))
                expr = f"{lit(*a)} in {p} of Interval[{lit(*b)}, {lit(*c)}]"
                rows.append((expr, {m.name: ge2(a, b, c, p, m) for m in MODELS}))
    return rows + build_hour_spread()


# ---------------------------------------------------------------------------
# Plan 025: hour values cut in two by an offset.
#
# A value is its span of instants, from its first to its last millisecond, as long
# as its precision. A pair with one offset is read in the digits written, and a
# pair with different offsets in the offset of the evaluation request; a value to
# the hour that that offset does not start on the hour is a spread, and is
# compared by the two ends of its span, which have every component. The
# comparison is truncated at the precision of the operator, the results of the
# corners are filled between the least and the greatest (truncating is
# monotonic and the span is continuous), and a predicate answers where it gives
# one answer for every result and nothing is unknown: a component the operator
# asks for that a value does not have.
# Read in the digits written where the precision is coarser than the hour.
# ---------------------------------------------------------------------------
EPOCH = datetime(1970, 1, 1)
PREC_MS = {3: 3600000, 4: 60000, 5: 1000, 6: 1}
SPAN_MS = {3: 3600000, 4: 60000, 5: 1000, 6: 1}


class V:
    """A DateTime with an offset: its wall clock, offset in minutes, and its
    precision as a component index (3 hour, 4 minute, 5 second)."""

    def __init__(self, wall, off, prec):
        self.wall, self.off, self.prec = wall, off, prec

    @property
    def first(self):  # the first instant, as a naive UTC time
        return self.wall - timedelta(minutes=self.off)

    @property
    def last(self):
        return self.first + timedelta(milliseconds=SPAN_MS[self.prec] - 1)

    def lit(self):
        text = {3: "%Y-%m-%dT%H", 4: "%Y-%m-%dT%H:%M", 5: "%Y-%m-%dT%H:%M:%S"}[self.prec]
        o = self.off
        return "@" + self.wall.strftime(text) + ("Z" if o == 0 else "%s%02d:%02d" % ("-" if o < 0 else "+", abs(o) // 60, abs(o) % 60))


def at(instant, off, prec):
    """The value that starts at (or, to the unit, contains) an instant, as the
    wall clock of an offset written to a precision."""
    wall = instant + timedelta(minutes=off)
    unit = {3: dict(minute=0, second=0, microsecond=0), 4: dict(second=0, microsecond=0), 5: dict(microsecond=0)}[prec]
    return V(wall.replace(**unit), off, prec)


def components(w):
    return (w.year, w.month, w.day, w.hour, w.minute, w.second, w.microsecond // 1000)


def frames_of(a, b, p, mode):
    """The offset each of a pair is read in, and whether a spread is told from a
    unit. 'rule' is the plan: written digits for one offset or a precision coarser
    than the hour, the offset of the evaluation otherwise. The other modes are
    how a reference reads: in the offset of an operand, or at UTC, a value to the
    hour that does not start on the hour read as its first instant's hour."""
    if mode in ("alwaysutc", "oldutc"):
        return 0, 0, False
    if p < 3 or a.off == b.off:
        return a.off, b.off, mode in ("rule", "frame", "js-rule")
    target = {"rule": EVAL, "frame": EVAL, "js-rule": EVAL, "left": a.off, "js-left": a.off,
              "right": b.off, "js-right": b.off, "utc": 0}[mode]
    return target, target, mode in ("rule", "frame", "js-rule")


def corners(v, frame, p, spans):
    """What to compare of a value read in a frame: (components, finest) for each
    corner of its span if it is a spread, else for the unit it was written to."""
    first = v.first + timedelta(minutes=frame)
    if spans and p >= 3 and v.prec == 3 and (first.minute or first.second or first.microsecond):
        last = v.last + timedelta(minutes=frame)
        return [(components(first), 6), (components(last), 6)]
    return [(components(first), v.prec)]


def fill(results):
    return set(range(min(results), max(results) + 1)) if results else set()


def results_of(a, b, p, mode="rule"):
    """The results a pair at precision p (an index) can come to, and whether any
    part of it is unknown."""
    fa, fb, spans = frames_of(a, b, p, mode)
    out, unknown = [], False
    for x, mx in corners(a, fa, p, spans):
        for y, my in corners(b, fb, p, spans):
            level = min(p, mx, my)
            cx, cy = x[:level + 1], y[:level + 1]
            if cx != cy:
                out.append((cx > cy) - (cx < cy))
            elif level < p:
                unknown = True
            else:
                out.append(0)
    return fill(out), unknown


def holds(res, test):
    results, unknown = res
    if unknown or not results:
        return None
    answers = {test(c) for c in results}
    return answers.pop() if len(answers) == 1 else None


def and3(x, y):
    if x is False or y is False:
        return False
    return None if x is None or y is None else True


def or3(x, y):
    if x is True or y is True:
        return True
    return None if x is None or y is None else False


POINT_TESTS = {"same": lambda c: c == 0, "sob": lambda c: c <= 0, "soa": lambda c: c >= 0,
               "before": lambda c: c < 0, "after": lambda c: c > 0}


def point3(op, a, b, p, mode="rule"):
    return holds(results_of(a, b, p, mode), POINT_TESTS[op])


def units(v, frame, p, spans):
    """The first and last unit at precision p the value can be in, as integers
    counting from 1970, and whether it is one."""
    first = v.first + timedelta(minutes=frame)
    last = v.last + timedelta(minutes=frame)
    step = PREC_MS[p]
    lo = int((first - EPOCH) / timedelta(milliseconds=1)) // step
    hi = int((last - EPOCH) / timedelta(milliseconds=1)) // step
    if not spans and v.prec == 3 and p >= 3:
        # read as its first instant's hour, a unit to the hour as written
        hi = lo if p == 3 else hi
    return lo, hi, lo == hi


def meets3(a, b, p, mode="rule"):
    """Whether b's unit can only be the one after a's."""
    fa, fb, spans = frames_of(a, b, p, mode)
    alo, ahi, aone = units(a, fa, p, spans)
    blo, bhi, bone = units(b, fb, p, spans)
    if ahi + 1 < blo or bhi < alo + 1:
        return False
    return True if aone and bone else None


def meets_either(A, B, p, mode="rule"):
    return or3(meets3(A[1], B[0], p, mode), meets3(B[1], A[0], p, mode))


def within3(x, lo, hi, p, mode="rule"):
    return and3(point3("sob", lo, x, p, mode), point3("sob", x, hi, p, mode))


# Where the other value is, from the first instant of the span, in seconds: before
# it, at its first instant, inside, at its last, just after it, well after it.
POSITIONS = [-4000, -1, 0, 1800, 3599, 3600, 4000]
# The hour values that are the spread: the instant they start at, which is the
# hour at Z and 05:30 to 06:29 at +05:30, and the same a day earlier so that the
# span crosses midnight at +05:30 (18:00Z is 23:30 there).
STARTS = [datetime(2020, 1, 1, 0, 0), datetime(2019, 12, 31, 18, 0)]
SPREAD_OFFSETS = [0, 60, 330, 345, -180, -570]
OTHER_OFFSETS = [0, 330, -180]
PLAN025 = "docs/plans/025-hour-spread-by-offset.md"

def guards():
    """Pairs that are not a spread, read as written: a precision coarser than the
    hour, one offset on both sides, and a value that the offset cuts on the hour."""
    z, m30, p1 = 0, 330, 60
    rows = [
        (V(datetime(2020, 1, 1, 0), z, 3), V(datetime(2019, 12, 31, 23, 0), -180, 4), "same", 2),
        (V(datetime(2020, 1, 1, 0), z, 3), V(datetime(2020, 1, 1, 9, 30), m30, 4), "same", 2),
        (V(datetime(2020, 1, 1, 0), z, 3), V(datetime(2020, 1, 1, 9, 30), m30, 4), "before", 2),
        (V(datetime(2020, 1, 1, 0), z, 3), V(datetime(2020, 1, 1, 0, 30), z, 4), "same", 4),
        (V(datetime(2020, 1, 1, 0), z, 3), V(datetime(2020, 1, 1, 1, 10), z, 4), "before", 3),
        (V(datetime(2020, 1, 1, 0), p1, 3), V(datetime(2020, 1, 1, 1, 30), p1, 4), "before", 4),
        (V(datetime(2020, 1, 1, 0), p1, 3), V(datetime(2020, 1, 1, 1, 30), p1, 4), "after", 4),
        (V(datetime(2020, 1, 1, 0), -570, 3), V(datetime(2020, 1, 1, 15, 30), z, 4), "same", 3),
        (V(datetime(2020, 1, 1, 0), -570, 3), V(datetime(2020, 1, 1, 15, 30), z, 4), "same", 4),
        (V(datetime(2020, 1, 1, 0), -570, 3), V(datetime(2020, 1, 1, 15, 30), z, 4), "before", 3),
    ]
    return rows


PHRASES = {"same": "same {u} as", "sob": "same {u} or before", "soa": "same {u} or after",
           "before": "before {u} of", "after": "after {u} of"}
UNIT_NAMES = {2: "day", 3: "hour", 4: "minute"}


def build_hour_spread():
    """The rows of plan 025: (expression, {model: answer}), the answer true, false
    or None for null."""
    rows = []
    modes = [m.name for m in MODELS]
    combos = [(PHRASES[op], op, p) for op in PHRASES for p in (3, 4)]
    n = 0
    # The pair of a spread and another value at each position, one operator and
    # precision for each (rotating through the ten), and the operands in each order.
    for off in SPREAD_OFFSETS:
        for start in STARTS:
            span = at(start, off, 3)
            for pos in POSITIONS:
                for prec in (3, 4, 5):
                    for ooff in OTHER_OFFSETS:
                        other = at(span.first + timedelta(seconds=pos), ooff, prec)
                        ph, op, p = combos[n % len(combos)]
                        a, b = (span, other) if n % 2 == 0 else (other, span)
                        unit = UNITS[p - 3]
                        expr = f"{a.lit()} {ph.format(u=unit)} {b.lit()}"
                        rows.append((expr, {m: point3(op, a, b, p, m) for m in modes}))
                        n += 1
    # A spread inside an interval, and an interval around a spread, with bounds at
    # the minute: the bounds are before, at the edges of, inside and after it.
    layouts = [(-7200, -4000), (-7200, 0), (-7200, 1800), (0, 7200), (1800, 7200),
               (3600, 7200), (-7200, 7200), (3700, 9000)]
    k = 0
    for off in SPREAD_OFFSETS:
        for start in STARTS:
            span = at(start, off, 3)
            for lo_s, hi_s in layouts:
                boff = [0, 330, -180][k % 3]
                lo = at(span.first + timedelta(seconds=lo_s), boff, 4)
                hi = at(span.first + timedelta(seconds=hi_s), boff, 4)
                for p in (3, 4):
                    unit = UNITS[p - 3]
                    expr = f"{span.lit()} in {unit} of Interval[{lo.lit()}, {hi.lit()}]"
                    rows.append((expr, {m: within3(span, lo, hi, p, m) for m in modes}))
                    expr = f"Interval[{lo.lit()}, {hi.lit()}] includes {unit} of {span.lit()}"
                    rows.append((expr, {m: within3(span, lo, hi, p, m) for m in modes}))
                k += 1
    # An interval that ends in a spread, or begins in one, against one that begins or
    # ends at each position: the successor of a span is not two whole hours.
    for off in SPREAD_OFFSETS:
        for start in STARTS:
            span = at(start, off, 3)
            before_span = at(span.first - timedelta(hours=1), off, 3)
            after_span = at(span.first + timedelta(hours=1), off, 3)
            for pos in (-4000, -1, 1800, 3600, 4000):
                for p in (3, 4):
                    unit = UNITS[p - 3]
                    boff = [0, 330][k % 2]
                    k += 1
                    nxt = at(span.first + timedelta(seconds=pos), boff, 4)
                    nxt2 = at(nxt.first + timedelta(hours=1), boff, 4)
                    A, B = (before_span, span), (nxt, nxt2)
                    expr = (f"Interval[{A[0].lit()}, {A[1].lit()}] meets {unit} of "
                            f"Interval[{B[0].lit()}, {B[1].lit()}]")
                    rows.append((expr, {m: meets_either(A, B, p, m) for m in modes}))
                    A, B = (nxt, nxt2), (span, after_span)
                    expr = (f"Interval[{A[0].lit()}, {A[1].lit()}] meets {unit} of "
                            f"Interval[{B[0].lit()}, {B[1].lit()}]")
                    rows.append((expr, {m: meets_either(A, B, p, m) for m in modes}))
    for a, b, op, p in guards():
        expr = f"{a.lit()} {PHRASES[op].format(u=UNIT_NAMES[p])} {b.lit()}"
        rows.append((expr, {m: point3(op, a, b, p, m) for m in modes}))
    return rows



def ge2(a, b, c, p, m):
    return compare(b, a, p, m) <= 0 and compare(a, c, p, m) <= 0


HEADER = """library Differential version '1.0'
using FHIR version '4.0.1'
include FHIRHelpers version '4.0.1'
parameter MP Interval<DateTime> default Interval[@2020-03-01, @2020-03-05]
parameter P DateTime default @2020-03-01
define function F(x DateTime): x
define function G(x Interval<DateTime>): x"""


PLAN = "docs/plans/024-same-precision-offset.md"
PLAN25 = "docs/plans/025-hour-spread-by-offset.md"
SPEC = ("CQL 1.5.3 (https://cql.hl7.org/N1A/09-b-cqlreference.html), Appendix B, Same As: \"When comparing DateTime "
        "values with different timezone offsets, implementations should normalize to the timezone offset of the "
        "evaluation request timestamp, but only when the comparison precision is hours, minutes, seconds, or "
        "milliseconds\"")
NO_OFFSET = re.compile(r"@\d{4}-\d\d-\d\dT\d\d(?::\d\d(?::\d\d(?:\.\d{3})?)?)?(?![\dZ+\-:.])")


ENGINE = {"java": "the Java engine", "js": "cql-execution"}


def left_reading(ref, refans, eng, name):
    return "deliberate", (
        f"decided in {PLAN} (decision of 14-11-2026, round 1, A1): a pair with different offsets is read in the "
        f"offset of the evaluation request, +05:30, where 00:10Z and 00:40Z are 05:40 and 06:10; {SPEC}; "
        f"{ENGINE[ref]} reads both in the offset of the left operand and answers {refans} ({name})")


def difference_in_hours(ref, refans, eng, name):
    return "deliberate", (
        f"decided in {PLAN} (the plan leaves difference in hours as it was, D4: it normalizes to the offset of the "
        f"evaluation request before it counts the hours, and {ENGINE[ref]} counts them between the instants "
        f"it truncates at their own offset); the point is the third of docs/hl7-cql-clarification.md, a draft to HL7 "
        f"on Duration and Difference; the answers are {refans} and {eng} ({name})")


def interval_error(ref, refans, eng, name):
    return "reference defect", (
        f"the Java engine refuses the interval `Interval[@2020-01-01T00Z, @2020-01-01T01:40+01:00]`, whose bounds are "
        f"00:00Z and 00:40Z, with `Invalid Interval - the ending boundary (@2020-01-01T01:40) must be greater than or "
        f"equal to the starting boundary (@2020-01-01T00)`, comparing a DateTime to the hour with one to the minute "
        f"on their written digits; cql-execution and this engine answer {eng} ({PLAN}, round 1, attack D9)")


def truncated_offset(ref, refans, eng, name):
    return "reference defect", (
        f"cql-execution places a DateTime with no offset at +05:00, the +05:30 of the evaluation request truncated to "
        f"whole hours by parseInt (lib/elm/literal.js, lib/elm/datetime.js); it answers {refans} where this engine, "
        f"and the Java engine, answer {eng} ({PLAN}, round 1, A3; the sentinel is TestCqlExecutionTruncatesAFractionalOffset)")


PROBE_VERDICTS = {
    "D1": left_reading, "D31": left_reading,
    "D4": difference_in_hours, "D25": difference_in_hours, "D34": difference_in_hours,
    "D27": interval_error, "D36": truncated_offset, "D37": truncated_offset,
}


def text(answer):
    return "null" if answer is None else "true" if answer else "false"


def verdicts(path):
    """Gives the divergences of the area their verdict, from what each model of
    the reading answers: the rule of the plan (which is this engine's, and the
    oracle's), the reading of each operand's offset, UTC, and the same with the
    +05:00 that cql-execution takes the +05:30 of the evaluation for."""
    names = [m.name for m in MODELS]
    answers = {f"D{i}": ans for i, (_, ans) in enumerate(rows, 1) if ans is not None}
    first_spread = len(rows) - len(build_hour_spread()) + 1
    exprs = {f"D{i}": e for i, (e, _) in enumerate(rows, 1)}
    out, counts = [], {}
    for line in open(path).read().split("\n"):
        if not line or line.startswith("#"):
            out.append(line)
            continue
        f = line.split("\t")
        name, ref = f[0], f[1]
        refans, eng = f[5], f[6]
        expr = exprs[name]
        if name in answers:
            model = {n: text(answers[name][n]) for n in names}
            assert model["rule"] == eng, (name, model, eng)
            hits = [n for n in names if model[n] == refans and n.startswith("js") == (ref == "js")]
            # A pair whose only written offset is the evaluation's has one offset
            # on both sides: what cql-execution answers apart is its truncation
            # alone, whichever reading it lands on (code review of plan 024,
            # round 1, finding 3).
            if int(name[1:]) < first_spread:
                assert hits, (name, refans, model)
            if int(name[1:]) >= first_spread:
                # The rows of plan 025, every offset written. The references read
                # a pair in the offset of an operand and a value to the hour as the
                # hour its first instant is in there; this engine reads a pair
                # with different offsets in the offset of the evaluation request
                # (plan 024) and a value to the hour that offset cuts in two as its
                # span (plan 025).
                reads = ", ".join(h for h in hits if h not in ("utc", "oldutc"))
                # Only `meets` at the minute with a bound to the hour is left without
                # a reading that gives the answer of the reference: it declines or
                # refuses to place a unit it has no minute for, where this engine
                # compares the units the span can be in (plan 025, successorOf).
                unplaced_unit = (f"{'the Java engine' if ref == 'java' else 'cql-execution'} answers {refans} "
                                 "for a bound to the hour at the minute, which no reading of this script gives")
                if not hits:
                    assert " meets minute of " in exprs[name], (name, refans, model)
                cls, ev = "deliberate", (
                    f"decided in {PLAN25} (decision D1 of 15-11-2026, the span of an hour that the offset cuts in two, "
                    f"read as the plan 008 reads it for `<` and `=`; and {PLAN} for a pair with different offsets, read "
                    f"in the offset of the evaluation request, +05:30); {name} is such a row; {SPEC}; "
                    f"{'the Java engine' if ref == 'java' else 'cql-execution'} reads both in the offset of an operand "
                    + (f"({reads}), a value to the hour as the hour its first instant is in, and answers {refans}"
                       if hits else unplaced_unit))
                counts[(ref, cls)] = counts.get((ref, cls), 0) + 1
                out.append("\t".join([name, ref, cls, eng, ev, refans, eng]))
                continue
            only_evaluation_offset = set(re.findall(r"[+-]\d\d:\d\d|Z", expr)) <= {"+05:30"}
            if ref == "js" and NO_OFFSET.search(expr) and ("js-rule" in hits or only_evaluation_offset):
                cls, ev = "reference defect", (
                    f"cql-execution places a DateTime with no offset at +05:00, the +05:30 of the evaluation request "
                    f"truncated to whole hours by parseInt (lib/elm/literal.js, lib/elm/datetime.js; plan 024, round 1, A3); "
                    f"read there it answers {refans}, and at +05:30, where the request is, {eng} ({PLAN})")
            else:
                reads = ", ".join(h for h in hits if h not in ("utc", "oldutc"))
                cls, ev = "deliberate", (
                    f"decided in {PLAN} (decision of 14-11-2026, round 1, A1): a pair with different offsets is read in the "
                    f"offset of the evaluation request, +05:30, and {name} is such a row; {SPEC}; the "
                    f"{'Java engine' if ref == 'java' else 'cql-execution'} reads both in the offset of an operand "
                    f"({reads}) and answers {refans}")
            counts[(ref, cls)] = counts.get((ref, cls), 0) + 1
            out.append("\t".join([name, ref, cls, eng, ev, refans, eng]))
        elif name in PROBE_VERDICTS:
            cls, ev = PROBE_VERDICTS[name](ref, refans, eng, name)
            counts[(ref, cls)] = counts.get((ref, cls), 0) + 1
            out.append("\t".join([name, ref, cls, eng, ev, refans, eng]))
        else:
            out.append(line)
    open(path, "w").write("\n".join(out))
    for k in sorted(counts):
        print(k, counts[k], file=sys.stderr)


rows = build()

if "--oracle" in sys.argv:
    for i, (_, ans) in enumerate(rows, 1):
        if ans is not None:
            print(f"D{i}\t" + ("null" if ans["rule"] is None else f"Boolean {'true' if ans['rule'] else 'false'}"))
    sys.exit(0)
if "--models" in sys.argv:
    names = [m.name for m in MODELS]
    print("name\t" + "\t".join(names))
    for i, (_, ans) in enumerate(rows, 1):
        if ans is not None:
            print(f"D{i}\t" + "\t".join(text(ans[n]) for n in names))
    sys.exit(0)

if "--verdicts" in sys.argv:
    verdicts(sys.argv[sys.argv.index("--verdicts") + 1])
    sys.exit(0)

print(HEADER)
print("""// Generated by scripts/fractionaloffset-area.py: the cases of
// docs/plans/024-same-precision-offset.md, evaluated at +05:30. First the probes
// of the plan and the attack of its adversarial round, then every operator with a
// precision of the hour to the millisecond over pairs of values with no offset,
// with one offset on both sides, with different offsets, and across a day.""")
for i, (expr, _) in enumerate(rows, 1):
    print(f"define D{i}: {expr}")
