'use strict';

// Runs each expression definition of an ELM library with cql-execution and
// records the value with its CQL type, in the same encoding as Evaluate.java,
// so that a second runtime can be diffed against this engine and against the
// first one.
//
// It runs the ELM the reference translator produced, so it is a second opinion
// on evaluation only: where it agrees with the Java engine, the translation they
// share has not been checked by either.
//
// usage: node evaluate.js <library.elm.json> <library.cql>
//        node evaluate.js --each <elms.json> <corpus.cql> <FHIRHelpers.elm.json> [data.json]
//
// With --each it runs a corpus Evaluate.java -each translated: the ELM of
// every definition, each a library of its own, over the same FHIR resources
// through cql-exec-fhir, with FHIRHelpers translated from the same text both
// other engines use.

const fs = require('fs');
const crypto = require('crypto');
const cql = require('cql-execution');
const cqlfhir = require('cql-exec-fhir');

// The same instant as Evaluate.java, at the same offset, for the same reason:
// at UTC an engine that defaulted to UTC instead of the evaluation offset would
// give the same answers, and the difference would not show.
const EVALUATION = '2026-01-01T00:00:00.000-03:00';

const SYSTEM = '{urn:hl7-org:elm-types:r1}';

async function main() {
  if (process.argv[2] === '--each') return each(process.argv.slice(3));
  const [elmPath, cqlPath] = process.argv.slice(2);
  if (!elmPath || !cqlPath) {
    console.error('usage: node evaluate.js <library.elm.json> <library.cql>');
    process.exit(2);
  }
  const elm = JSON.parse(fs.readFileSync(elmPath, 'utf8'));
  const library = new cql.Library(elm);
  const executionDateTime = cql.DateTime.parse(EVALUATION);

  const results = {};
  for (const def of elm.library.statements.def) {
    if (def.type === 'FunctionDef' || def.name === 'Patient') continue;
    let entry;
    try {
      // A context per definition, so that one that throws is recorded and the
      // rest still run. Executor.exec evaluates them all in one context and
      // one failure loses every answer.
      const ctx = new cql.UnfilteredContext(library, new cql.Results(), undefined, {}, executionDateTime);
      const value = await library.expressions[def.name].exec(ctx);
      entry = encode(value, typeOf(def));
    } catch (e) {
      entry = { error: e.constructor.name, message: String(e.message) };
    }
    results[def.name] = entry;
  }

  const out = {
    library: elm.library.identifier.id,
    sourceSha256: crypto.createHash('sha256').update(fs.readFileSync(cqlPath)).digest('hex'),
    engine: 'cql-execution ' + require('cql-execution/package.json').version,
    evaluationDateTime: EVALUATION,
    results,
  };
  process.stdout.write(JSON.stringify(out, null, 2) + '\n');
}

// each runs the definitions of a corpus over FHIR data, each in its own library,
// for the patient of data.json: a Patient and lists of resources by type, the
// file the snapshot of go-cql serves. A definition the translator refused has no
// ELM, and Evaluate.java has its error; it is not run here.
async function each([elmsPath, cqlPath, helpersPath, dataPath]) {
  if (!elmsPath || !cqlPath || !helpersPath) {
    console.error('usage: node evaluate.js --each <elms.json> <corpus.cql> <FHIRHelpers.elm.json> [data.json]');
    process.exit(2);
  }
  const elms = JSON.parse(fs.readFileSync(elmsPath, 'utf8'));
  const helpers = JSON.parse(fs.readFileSync(helpersPath, 'utf8'));
  const executionDateTime = cql.DateTime.parse(EVALUATION);
  let bundle = null;
  let patientId = null;
  if (dataPath) {
    const data = JSON.parse(fs.readFileSync(dataPath, 'utf8'));
    const entry = [];
    for (const [kind, v] of Object.entries(data)) {
      for (const resource of Array.isArray(v) ? v : [v]) {
        entry.push({ resource });
        if (kind === 'Patient') patientId = resource.id;
      }
    }
    bundle = { resourceType: 'Bundle', type: 'collection', entry };
  }
  const results = {};
  // cql-execution rejects a promise of its own outside the one awaited for
  // some expressions — Multiply over a null operand — and the process would
  // die with the answers half written. Such a rejection belongs to the
  // definition being run, the only one in flight, and is its answer.
  let current = null;
  process.on('unhandledRejection', (e) => {
    if (current) results[current] = { error: e.constructor.name, message: String(e.message) };
  });
  for (const [name, elm] of Object.entries(elms)) {
    current = name;
    let entry;
    try {
      const library = new cql.Library(elm, new cql.Repository({ FHIRHelpers: helpers }));
      const def = elm.library.statements.def.find((d) => d.name === name);
      let value;
      if (bundle) {
        // A source per definition: an execution walks the patients of its
        // source, and the next one would find none left.
        const source = cqlfhir.PatientSource.FHIRv401();
        source.loadBundles([bundle]);
        const out = await new cql.Executor(library).exec(source, executionDateTime);
        value = out.patientResults[patientId][name];
      } else {
        const ctx = new cql.UnfilteredContext(library, new cql.Results(), undefined, {}, executionDateTime);
        value = await library.expressions[name].exec(ctx);
      }
      entry = encode(value, typeOf(def));
    } catch (e) {
      entry = { error: e.constructor.name, message: String(e.message) };
    }
    // Let a rejection the run left behind land before the answer is kept.
    await new Promise((resolve) => setImmediate(resolve));
    if (!(name in results)) results[name] = entry;
  }
  current = null;
  const out = {
    library: 'corpus ' + cqlPath.split('/').pop(),
    sourceSha256: crypto.createHash('sha256').update(fs.readFileSync(cqlPath)).digest('hex'),
    engine: 'cql-execution ' + require('cql-execution/package.json').version +
      ', cql-exec-fhir ' + require('cql-exec-fhir/package.json').version,
    evaluationDateTime: EVALUATION,
    results,
  };
  process.stdout.write(JSON.stringify(out, null, 2) + '\n');
}

// typeOf reads the static type the translator annotated a definition with.
// cql-execution keeps Integer and Decimal as the same JavaScript number, so the
// runtime value cannot tell them apart and the annotation is the only source.
// That makes the number's type the translator's, not this runtime's — said
// here because it is the one field this encoding does not observe.
function typeOf(node) {
  if (node.resultTypeSpecifier) return node.resultTypeSpecifier;
  if (node.resultTypeName) return { type: 'NamedTypeSpecifier', name: node.resultTypeName };
  return null;
}

function elementTypeOf(spec) {
  return spec && spec.type === 'ListTypeSpecifier' ? spec.elementType : null;
}

function nameOf(spec) {
  return spec && spec.type === 'NamedTypeSpecifier' ? spec.name.replace(SYSTEM, '') : null;
}

function encode(v, spec) {
  if (v === null || v === undefined) return null;
  // A FHIR value from the data: a resource, an element or a primitive, by its
  // type in the model and its JSON, as Evaluate.java writes one.
  if (typeof v.getTypeInfo === 'function' && v.getTypeInfo() != null) {
    const name = 'FHIR.' + v.getTypeInfo().name.replace(/^FHIR\./, '');
    const json = v._json;
    // A primitive comes wrapped, { value: 'male' }, beside its id and
    // extensions; Evaluate.java writes the value itself. It is told by that
    // shape rather than by its type's name, since a binding names it here —
    // FHIR.AdministrativeGender where the Java engine says FHIR.code.
    if (json !== null && typeof json === 'object' && 'value' in json &&
        Object.keys(json).every((k) => ['value', 'id', 'extension'].includes(k))) {
      return { type: name, value: String(json.value) };
    }
    return { type: name, value: json !== null && typeof json === 'object' ? JSON.stringify(json) : String(json) };
  }
  if (typeof v === 'boolean') return { type: 'Boolean', value: v };
  // A Long comes back as text or a bigint, not a number, so the static type is
  // read before the JavaScript one.
  if (nameOf(spec) === 'Long' && ['string', 'number', 'bigint'].includes(typeof v)) {
    return { type: 'Long', value: String(v) };
  }
  if (typeof v === 'string') return { type: 'String', value: v };
  if (typeof v === 'number') {
    const name = nameOf(spec);
    if (name === 'Integer') return { type: 'Integer', value: String(v) };
    if (name === 'Decimal') return { type: 'Decimal', value: decimalText(v) };
    // A number with no static type to read: left visible, not guessed at.
    return { type: 'Unencoded:number', value: String(v) };
  }
  if (Array.isArray(v)) {
    const el = elementTypeOf(spec);
    return { type: 'List', elements: v.map((x) => encode(x, el)) };
  }
  if (v.isQuantity) return { type: 'Quantity', value: decimalText(v.value), unit: v.unit };
  // Before isDateTime: cql-execution represents a Time as a DateTime with no
  // date, and its isDateTime is true too.
  if (typeof v.isTime === 'function' && v.isTime()) {
    return { type: 'Time', value: v.toString(), precision: v.getPrecision() };
  }
  if (v.isDateTime) return encodeDateTime(v);
  if (v.isDate) return { type: 'Date', value: v.toString(), precision: v.getPrecision() };
  if (v.isInterval) {
    const point = spec && spec.type === 'IntervalTypeSpecifier' ? spec.pointType : null;
    return {
      type: 'Interval',
      low: encode(v.low, point), lowClosed: v.lowClosed,
      high: encode(v.high, point), highClosed: v.highClosed,
    };
  }
  // A tuple is a plain object there. Only a value the translator typed as a
  // tuple is encoded as one: any other plain object stays Unencoded, rather
  // than passing for a tuple and hiding what it is. Its fields are written
  // sorted, as the Java encoding writes them, each with its declared type.
  if (Object.getPrototypeOf(v) === Object.prototype && spec && spec.type === 'TupleTypeSpecifier') {
    const declared = {};
    for (const el of spec.element || []) declared[el.name] = el.elementType || el.type;
    const fields = {};
    for (const k of Object.keys(v).sort()) fields[k] = encode(v[k], declared[k] || null);
    return { type: 'Tuple', fields };
  }
  return { type: 'Unencoded:' + (v.constructor ? v.constructor.name : typeof v), value: String(v) };
}

// decimalText writes a Decimal as cql-execution holds it. It computes in binary
// floating point, so 0.1 + 0.2 is 0.30000000000000004 there, and that is what
// is recorded. Rounding it to the eight places of the CQL scale would put an
// answer in its mouth: 10.0 / 3.0 = 3.33333333 is false in cql-execution, and a
// rounded 3.33333333 would have counted as agreeing with the Java engine.
function decimalText(n) {
  return String(n);
}

// encodeDateTime writes the instant in full ISO 8601 whatever the precision,
// filling what the value does not state from the start of its span, as
// Evaluate.java does through java.time. The value field is for reading.
function encodeDateTime(d) {
  const pad = (n, w) => String(n).padStart(w, '0');
  const offset = d.timezoneOffset == null ? null : d.timezoneOffset;
  let instant = null;
  if (offset != null) {
    const sign = offset < 0 ? '-' : '+';
    const minutes = Math.round(Math.abs(offset) * 60);
    const zone = minutes === 0 ? 'Z' : `${sign}${pad(Math.floor(minutes / 60), 2)}:${pad(minutes % 60, 2)}`;
    const ms = d.millisecond ? `.${pad(d.millisecond, 3)}` : '';
    instant = `${pad(d.year, 4)}-${pad(d.month || 1, 2)}-${pad(d.day || 1, 2)}` +
      `T${pad(d.hour || 0, 2)}:${pad(d.minute || 0, 2)}:${pad(d.second || 0, 2)}${ms}${zone}`;
  }
  return { type: 'DateTime', value: d.toString(), precision: d.getPrecision(), instant };
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
