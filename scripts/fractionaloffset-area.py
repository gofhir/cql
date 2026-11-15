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


def verdicts(path):
    """Gives the divergences of the area their verdict, from what each model of
    the reading answers: the rule of the plan (which is this engine's, and the
    oracle's), the reading of each operand's offset, UTC, and the same with the
    +05:00 that cql-execution takes the +05:30 of the evaluation for."""
    names = [m.name for m in MODELS]
    answers = {f"D{i}": ans for i, (_, ans) in enumerate(rows, 1) if ans is not None}
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
            model = {n: "true" if answers[name][n] else "false" for n in names}
            assert model["rule"] == eng, (name, model, eng)
            hits = [n for n in names if model[n] == refans and n.startswith("js") == (ref == "js")]
            assert hits, (name, refans, model)
            # A pair whose only written offset is the evaluation's has one offset
            # on both sides: what cql-execution answers apart is its truncation
            # alone, whichever reading it lands on (code review of plan 024,
            # round 1, finding 3).
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
            print(f"D{i}\tBoolean {'true' if ans['rule'] else 'false'}")
    sys.exit(0)
if "--models" in sys.argv:
    names = [m.name for m in MODELS]
    print("name\t" + "\t".join(names))
    for i, (_, ans) in enumerate(rows, 1):
        if ans is not None:
            print(f"D{i}\t" + "\t".join('true' if ans[n] else 'false' for n in names))
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
