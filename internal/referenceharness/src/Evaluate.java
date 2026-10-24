import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import ca.uhn.fhir.context.BaseRuntimeChildDefinition;
import ca.uhn.fhir.context.BaseRuntimeElementCompositeDefinition;
import ca.uhn.fhir.context.BaseRuntimeElementDefinition;
import ca.uhn.fhir.context.RuntimeResourceBlockDefinition;
import ca.uhn.fhir.context.FhirContext;
import ca.uhn.fhir.parser.IParser;
import com.fasterxml.jackson.databind.JsonNode;
import org.cqframework.cql.cql2elm.*;
import org.hl7.elm.r1.ExpressionDef;
import org.hl7.elm.r1.FunctionDef;
import org.hl7.elm.r1.Library;
import org.hl7.elm.r1.VersionedIdentifier;
import org.hl7.fhir.instance.model.api.IBase;
import org.hl7.fhir.instance.model.api.IBaseElement;
import org.hl7.fhir.instance.model.api.IBaseResource;
import org.hl7.fhir.instance.model.api.IPrimitiveType;
import org.hl7.fhir.r4.model.Base;
import org.hl7.fhir.r4.model.PrimitiveType;
import org.opencds.cqf.cql.engine.data.CompositeDataProvider;
import org.opencds.cqf.cql.engine.data.DataProvider;
import org.opencds.cqf.cql.engine.execution.CqlEngine;
import org.opencds.cqf.cql.engine.execution.Environment;
import org.opencds.cqf.cql.engine.execution.EvaluationExpressionRef;
import org.opencds.cqf.cql.engine.execution.EvaluationParams;
import org.opencds.cqf.cql.engine.fhir.model.R4FhirModelResolver;
import org.opencds.cqf.cql.engine.retrieve.RetrieveProvider;
// Not runtime.*: 5.x has its own Boolean, Integer, Long, String and List, which
// clash with java.lang and java.util; those are written out where they are used.
import org.opencds.cqf.cql.engine.runtime.ClassInstance;
import org.opencds.cqf.cql.engine.runtime.Code;
import org.opencds.cqf.cql.engine.runtime.Concept;
import org.opencds.cqf.cql.engine.runtime.Date;
import org.opencds.cqf.cql.engine.runtime.DateTime;
import org.opencds.cqf.cql.engine.runtime.Interval;
import org.opencds.cqf.cql.engine.runtime.Quantity;
import org.opencds.cqf.cql.engine.runtime.Time;
import org.opencds.cqf.cql.engine.runtime.Tuple;
import org.opencds.cqf.cql.engine.runtime.Value;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.regex.Matcher;
import java.util.regex.Pattern;
import java.security.MessageDigest;
import java.util.HexFormat;
import java.nio.file.*;
import java.time.ZonedDateTime;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Map;
import java.util.TreeMap;

/** Ejecuta cada definición con el engine de referencia y registra el valor
 *  tipado, para que lo que el motor responde se compare contra un resultado
 *  y no contra lo que uno cree que la spec dice.
 *
 *  La salida no pasa por nada de go-cql: un oráculo que usara su String() o su
 *  igualdad heredaría los mismos defectos que tiene que encontrar. */
public class Evaluate {
    // 5.x logs through kotlin-logging, which prints a startup line to stdout, where
    // the JSON goes. Turned off before any class of the library loads it.
    static {
        io.github.oshai.kotlinlogging.KotlinLoggingConfiguration.INSTANCE.setLogStartupMessage(false);
    }

    // Fijo, para que Now() y los DateTime sin offset no dependan de la máquina. No
    // es UTC a propósito: un motor que tomara UTC por defecto en vez del offset de
    // la evaluación respondería lo mismo, y la diferencia no se vería.
    static final ZonedDateTime EVALUATION = ZonedDateTime.parse("2026-01-01T00:00:00-03:00");

    public static void main(String[] args) throws Exception {
        if (args.length > 0 && args[0].equals("-each")) {
            each(Arrays.copyOfRange(args, 1, args.length));
            return;
        }
        if (args.length < 1) {
            System.err.println("uso: Evaluate <fichero.cql> [dir-de-librerias]");
            System.err.println("     Evaluate -each <corpus.cql> <dir-de-librerias> [datos.json] [elm-salida.json]");
            System.exit(2);
        }
        Path file = Paths.get(args[0]);
        Path dir = args.length > 1 ? Paths.get(args[1]) : file.getParent();
        String text = Files.readString(file);

        ModelManager models = new ModelManager();
        LibraryManager libs = new LibraryManager(models);
        // El texto va primero: el engine resuelve la raíz por su identificador,
        // y el nombre del fichero no tiene por qué coincidir con él.
        libs.getLibrarySourceLoader().registerProvider(new StringLibrarySourceProvider(List.of(text)));
        libs.getLibrarySourceLoader().registerProvider(new DefaultLibrarySourceProvider(kotlinx.io.files.PathsJvmKt.Path(dir.toString())));
        libs.getCqlCompilerOptions().setOptions(
                CqlCompilerOptions.Options.EnableResultTypes,
                CqlCompilerOptions.Options.EnableLocators);

        CqlTranslator t = CqlTranslator.fromText(text, libs);
        if (!t.getErrors().isEmpty()) {
            // Igual que Translate: sin salida, para que un fallo de traducción
            // nunca quede registrado como referencia.
            for (Object e : t.getErrors()) System.err.println("ERROR: " + e);
            System.exit(1);
        }
        Library lib = t.toELM();

        CqlEngine engine = new CqlEngine(new Environment(libs));
        Map<String, Object> results = new LinkedHashMap<>();
        for (ExpressionDef def : lib.getStatements().getDef()) {
            if (def instanceof FunctionDef || def.getName().equals("Patient")) continue;
            Object entry;
            try {
                entry = encode(evaluate(engine, lib.getIdentifier(), def.getName(), null));
            } catch (Exception | StackOverflowError e) {
                // Un error es una respuesta: se registra, no aborta el resto. Una
                // recursión sin fondo también, aunque Java no la cuente como Exception.
                Map<String, Object> err = new LinkedHashMap<>();
                err.put("error", e.getClass().getSimpleName());
                err.put("message", message(e.getMessage()));
                entry = err;
            }
            results.put(def.getName(), entry);
        }

        Map<String, Object> out = new LinkedHashMap<>();
        out.put("library", lib.getIdentifier().getId());
        // El hash ata la referencia al texto exacto: una definición editada sin
        // regenerar se detecta, en vez de leerse como una divergencia del motor.
        out.put("sourceSha256", HexFormat.of().formatHex(
                MessageDigest.getInstance("SHA-256").digest(Files.readAllBytes(file))));
        out.put("engine", org.cqframework.engine.BuildConfig.IMPLEMENTATION_VERSION);
        // ISO completo: toString() omite los segundos cuando son cero.
        out.put("evaluationDateTime", EVALUATION.format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME));
        out.put("results", results);
        System.out.println(new ObjectMapper().enable(SerializationFeature.INDENT_OUTPUT).writeValueAsString(out));
    }

    /** A Java object's identity after its qualified class name, as
     *  org.hl7.elm.r1.NamedTypeSpecifier@21e57990, a nested class's or an array's.
     *  A CQL date, @2020-01-01, is none: the hash is not followed by a word
     *  character or a hyphen. TestJavaIdentityReadsOnlyAnIdentity holds the same
     *  pattern. */
    static final Pattern IDENTITY = Pattern.compile("((?:[a-z][a-z0-9]*\\.)+[A-Z][\\w$]*;?)@[0-9a-f]{1,8}(?![\\w-])");

    /** A message as the references record it: 5.x writes the identity of some
     *  objects in its messages, a hash of the run and not of the answer, and the
     *  two runs of the harness would differ. Only the identity goes; the class
     *  name stays, and the error's class is not touched. */
    static String message(String m) {
        return IDENTITY.matcher(String.valueOf(m)).replaceAll("$1@…");
    }

    /** A failure of the harness itself, not an answer of the engine: an Error, so
     *  that the handlers that record an engine's exception as its answer do not
     *  catch it, and the harness exits with it instead of writing a reference. */
    static final class EncodingError extends Error {
        EncodingError(String message) { super(message); }
    }

    /** Evaluates one definition of one library. 5.x takes EvaluationParams and keeps
     *  each library's error aside; getOnlyResultOrThrow throws it, as 3.26.0's
     *  single-library evaluate did, so an error is still caught as one. */
    static Value evaluate(CqlEngine engine, VersionedIdentifier id, String name, kotlin.Pair<String, String> context) {
        EvaluationParams.Builder params = new EvaluationParams.Builder();
        params.library(id, new EvaluationParams.LibraryParams(List.of(new EvaluationExpressionRef(name))));
        params.setContextParameter(context);
        params.setEvaluationDateTime(EVALUATION);
        return engine.evaluate(params.build()).getOnlyResultOrThrow().get(name).getValue();
    }

    /** The model resolver of the data provider. 5.x hands the engine FHIR data as
     *  ClassInstance values, not HAPI objects, and encode() turns them back with it. */
    static final R4FhirModelResolver FHIR = new R4FhirModelResolver();

    /** Una definición generada por línea; lo que va antes de la primera es el
     *  encabezado del área: la librería, su preámbulo. */
    static final Pattern DEFINE = Pattern.compile("(?m)^define (D\\d+):.*$");

    /** Modo corpus: cada definición generada se traduce y evalúa en una librería
     *  propia, con el encabezado del área. El corpus contiene expresiones inválidas
     *  a propósito, y el traductor rechaza el fichero entero ante la primera; así
     *  falla sola, y su error de traducción es su respuesta.
     *
     *  Cada una lleva su propio identificador (D17), para que el LibraryManager
     *  compile FHIRHelpers una vez y no novecientas.
     *
     *  Con datos.json, las definiciones se evalúan sobre recursos FHIR: el mismo
     *  fichero que sirve el snapshot de go-cql, un Patient de contexto y listas de
     *  recursos por tipo, o "-" para evaluar sin datos. Con elm-salida.json,
     *  escribe además el ELM de cada una,
     *  para que cql-execution evalúe exactamente la misma traducción. */
    static void each(String[] args) throws Exception {
        if (args.length < 2) {
            System.err.println("uso: Evaluate -each <corpus.cql> <dir-de-librerias> [datos.json] [elm-salida.json]");
            System.exit(2);
        }
        Path file = Paths.get(args[0]);
        String text = Files.readString(file);
        Matcher first = DEFINE.matcher(text);
        if (!first.find()) {
            System.err.println("ERROR: el corpus no tiene definiciones generadas");
            System.exit(1);
        }
        String header = text.substring(0, first.start());

        Map<String, String> sources = new HashMap<>();
        ModelManager models = new ModelManager();
        LibraryManager libs = new LibraryManager(models);
        libs.getLibrarySourceLoader().registerProvider(new LibrarySourceProvider() {
            @Override
            public kotlinx.io.Source getLibrarySource(VersionedIdentifier id) {
                String src = sources.get(id.getId());
                if (src == null) return null;
                // 5.x reads a source through kotlinx-io instead of an InputStream.
                kotlinx.io.Buffer b = new kotlinx.io.Buffer();
                byte[] bytes = src.getBytes(StandardCharsets.UTF_8);
                b.write(bytes, 0, bytes.length);
                return b;
            }
        });
        libs.getLibrarySourceLoader().registerProvider(new DefaultLibrarySourceProvider(kotlinx.io.files.PathsJvmKt.Path(args[1])));
        libs.getCqlCompilerOptions().setOptions(
                CqlCompilerOptions.Options.EnableResultTypes,
                CqlCompilerOptions.Options.EnableLocators);

        Map<String, DataProvider> data = new HashMap<>();
        kotlin.Pair<String, String> context = null;
        // "-" for datos.json evaluates without data, so that elm-salida.json can
        // still be asked for.
        if (args.length > 2 && !args[2].equals("-")) {
            FhirBundle bundle = FhirBundle.read(Paths.get(args[2]));
            data.put("http://hl7.org/fhir", new CompositeDataProvider(FHIR, bundle));
            context = new kotlin.Pair<>("Patient", bundle.patientId);
        }
        CqlEngine engine = new CqlEngine(new Environment(libs, data, null));

        Map<String, Object> results = new LinkedHashMap<>();
        Map<String, Object> elms = new LinkedHashMap<>();
        Matcher m = DEFINE.matcher(text);
        while (m.find()) {
            String name = m.group(1);
            String src = header.replaceFirst("(?m)^library \\S+", "library " + name) + m.group() + "\n";
            sources.put(name, src);
            CqlTranslator t = CqlTranslator.fromText(src, libs);
            if (!t.getErrors().isEmpty()) {
                Map<String, Object> err = new LinkedHashMap<>();
                err.put("error", "Translation");
                err.put("message", message(t.getErrors().get(0).getMessage()));
                results.put(name, err);
                continue;
            }
            Library lib = t.toELM();
            elms.put(name, new ObjectMapper().readTree(t.toJson()));
            Object entry;
            try {
                entry = encode(evaluate(engine, lib.getIdentifier(), name, context));
            } catch (Exception | StackOverflowError e) {
                Map<String, Object> err = new LinkedHashMap<>();
                err.put("error", e.getClass().getSimpleName());
                err.put("message", message(e.getMessage()));
                entry = err;
            }
            results.put(name, entry);
        }
        if (args.length > 3) {
            Files.writeString(Paths.get(args[3]), new ObjectMapper().writeValueAsString(elms));
        }

        Map<String, Object> out = new LinkedHashMap<>();
        out.put("library", "corpus " + file.getFileName());
        out.put("sourceSha256", HexFormat.of().formatHex(
                MessageDigest.getInstance("SHA-256").digest(Files.readAllBytes(file))));
        out.put("engine", org.cqframework.engine.BuildConfig.IMPLEMENTATION_VERSION);
        out.put("evaluationDateTime", EVALUATION.format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME));
        out.put("results", results);
        System.out.println(new ObjectMapper().enable(SerializationFeature.INDENT_OUTPUT).writeValueAsString(out));
    }

    /** Los recursos de datos.json, servidos por tipo. Sin filtros de código ni de
     *  fecha: un retrieve devuelve todos los de su tipo, y es la expresión la que
     *  filtra, como en el proveedor del snapshot de go-cql. */
    static final class FhirBundle implements RetrieveProvider {
        final Map<String, List<Value>> byType = new HashMap<>();
        String patientId;

        static FhirBundle read(Path path) throws Exception {
            IParser parser = FhirContext.forR4Cached().newJsonParser();
            ObjectMapper json = new ObjectMapper();
            FhirBundle b = new FhirBundle();
            JsonNode root = json.readTree(Files.readString(path));
            root.fields().forEachRemaining(e -> {
                List<JsonNode> items = new ArrayList<>();
                if (e.getValue().isArray()) e.getValue().forEach(items::add); else items.add(e.getValue());
                for (JsonNode item : items) {
                    IBaseResource r = parser.parseResource(item.toString());
                    // 5.x serves values, not HAPI resources: each one is converted once, as
                    // the engine's own FHIR retrieve providers do with toCqlValue.
                    b.byType.computeIfAbsent(e.getKey(), k -> new ArrayList<>()).add(FHIR.toCqlValue(r, false));
                    if (e.getKey().equals("Patient")) b.patientId = r.getIdElement().getIdPart();
                }
            });
            return b;
        }

        @Override
        public Iterable<Value> retrieve(String context, String contextPath, String contextValue, String dataType,
                String templateId, String codePath, Iterable<Code> codes, String valueSet, String datePath,
                String dateLowPath, String dateHighPath, Interval dateRange) {
            return byType.getOrDefault(dataType, List.of());
        }
    }

    /** Codifica un valor con su tipo CQL explícito. null es JSON null; una lista
     *  vacía sigue siendo una lista, para no confundir las dos. Los números van
     *  como texto, para que 1 y 1.0 no se vuelvan iguales al serializar. */
    static Object encode(Value v) {
        if (v == null) return null;
        Map<String, Object> m = new LinkedHashMap<>();
        if (v instanceof org.opencds.cqf.cql.engine.runtime.Boolean b) {
            m.put("type", "Boolean"); m.put("value", b.getValue());
        } else if (v instanceof org.opencds.cqf.cql.engine.runtime.Integer i) {
            // 5.x's Long.toString() appends an L; the number is its value.
            m.put("type", "Integer"); m.put("value", java.lang.Integer.toString(i.getValue()));
        } else if (v instanceof org.opencds.cqf.cql.engine.runtime.Long l) {
            m.put("type", "Long"); m.put("value", java.lang.Long.toString(l.getValue()));
        } else if (v instanceof org.opencds.cqf.cql.engine.runtime.Decimal d) {
            m.put("type", "Decimal"); m.put("value", d.getValue().toPlainString());
        } else if (v instanceof org.opencds.cqf.cql.engine.runtime.String s) {
            m.put("type", "String"); m.put("value", s.getValue());
        } else if (v instanceof Quantity q) {
            m.put("type", "Quantity"); m.put("value", q.getValue().toPlainString()); m.put("unit", q.getUnit());
        } else if (v instanceof DateTime d) {
            // toString() es para leerlo: omite el offset en unas precisiones y lo
            // agrega en otras. Lo que se compara es instant, ISO completo siempre.
            // 5.x's toString() prefixes the CQL @; toStringInner() is 3.26.0's toString().
            m.put("type", "DateTime"); m.put("value", d.toStringInner()); m.put("precision", d.getPrecision().toString());
            m.put("offset", String.valueOf(d.getZoneOffset()));
            m.put("instant", d.getDateTime().format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME));
        } else if (v instanceof Date d) {
            m.put("type", "Date"); m.put("value", d.toStringInner()); m.put("precision", d.getPrecision().toString());
        } else if (v instanceof Time d) {
            m.put("type", "Time"); m.put("value", d.toStringInner()); m.put("precision", d.getPrecision().toString());
        } else if (v instanceof Interval i) {
            m.put("type", "Interval");
            m.put("low", encode(i.getLow())); m.put("lowClosed", i.getLowClosed());
            m.put("high", encode(i.getHigh())); m.put("highClosed", i.getHighClosed());
        } else if (v instanceof Code c) {
            // Every field is written, null included: an absent field and a null
            // one are one value, and an empty one is another.
            m.put("type", "Code"); m.put("code", c.getCode()); m.put("system", c.getSystem());
            m.put("version", c.getVersion()); m.put("display", c.getDisplay());
        } else if (v instanceof Concept co) {
            List<Object> codes = null;
            if (co.getCodes() != null) {
                codes = new ArrayList<>();
                for (Code c : co.getCodes()) codes.add(encode(c));
            }
            m.put("type", "Concept"); m.put("codes", codes); m.put("display", co.getDisplay());
        } else if (v instanceof Tuple tu) {
            Map<String, Object> el = new TreeMap<>();
            tu.getElements().forEach((k, x) -> el.put(k, encode(x)));
            m.put("type", "Tuple"); m.put("fields", el);
        } else if (v instanceof ClassInstance ci && ci.getType().getNamespaceURI().equals("http://hl7.org/fhir")) {
            // 5.x hands FHIR data over as ClassInstance. It goes back to its HAPI form,
            // so that it is written exactly as 3.26.0 wrote the HAPI object it got,
            // under the type 5.x gives it: FHIR.AdministrativeGender, where HAPI
            // says code. One the harness cannot rebuild stops it (EncodingError).
            return encodeHapi(toHapi(ci), "FHIR." + ci.getType().getLocalPart());
        } else if (v instanceof org.opencds.cqf.cql.engine.runtime.ValueSet vs) {
            // The CQL System ValueSet by its fields, as a tuple's are written.
            List<Object> systems = new ArrayList<>();
            for (org.opencds.cqf.cql.engine.runtime.CodeSystem cs : vs.getCodeSystems()) systems.add(encode(cs));
            Map<String, Object> el = new TreeMap<>();
            el.put("id", vs.getId()); el.put("version", vs.getVersion()); el.put("name", vs.getName());
            // A LinkedHashMap, as every map written: Map.of's order changes from one
            // run to the next.
            Map<String, Object> list = new LinkedHashMap<>();
            list.put("type", "List"); list.put("elements", systems);
            el.put("codesystems", list);
            m.put("type", "ValueSet"); m.put("fields", el);
        } else if (v instanceof org.opencds.cqf.cql.engine.runtime.CodeSystem cs) {
            Map<String, Object> el = new TreeMap<>();
            el.put("id", cs.getId()); el.put("version", cs.getVersion()); el.put("name", cs.getName());
            m.put("type", "CodeSystem"); m.put("fields", el);
        } else if (v instanceof Iterable<?> it) {
            List<Object> el = new ArrayList<>();
            for (Object x : it) el.add(encode((Value) x));
            m.put("type", "List"); m.put("elements", el);
        } else {
            // Sin codificación propia: se deja visible en vez de adivinarla.
            m.put("type", "Unencoded:" + v.getClass().getName()); m.put("value", v.toString());
        }
        return m;
    }

    /** The FHIR branches of 3.26.0's encode, over the HAPI object, under the
     *  type the engine gave it. */
    static Object encodeHapi(Object v, String type) {
        Map<String, Object> m = new LinkedHashMap<>();
        if (v instanceof PrimitiveType<?> p) {
            // A FHIR primitive, not a System one: its value is the text the
            // resource holds. Its extensions are reached through it, as elements.
            m.put("type", type); m.put("value", p.getValueAsString());
        } else if (v instanceof Base b) {
            // A FHIR resource or element: its JSON.
            m.put("type", type);
            m.put("value", b instanceof IBaseResource r
                    ? FhirContext.forR4Cached().newJsonParser().encodeResourceToString(r)
                    : FhirContext.forR4Cached().newJsonParser().encodeToString((org.hl7.fhir.instance.model.api.IBase) b));
        } else {
            throw new EncodingError("HAPI " + v.getClass().getName());
        }
        return m;
    }

    /** A FHIR ClassInstance back to the HAPI object FhirModelResolver.toCqlValue made
     *  it from: the class by its type name, each child by its element name, a
     *  primitive's value by the resolver's own setter, and its extensions. An
     *  element it does not rebuild stops the harness: written without it, the
     *  value would not be the one the engine answered. */
    static IBase toHapi(ClassInstance ci) {
        return toHapi(ci, FHIR.createHapiInstance(ci.getType().getLocalPart()));
    }

    /** toHapi into the instance o, made for it: a backbone element's class is its
     *  parent's (Claim.PayeeComponent), and its type's name alone, PayeeComponent,
     *  names another resource's as well. */
    static IBase toHapi(ClassInstance ci, Object o) {
        Map<String, Value> el = ci.getElements();
        java.util.Set<String> rebuilt = new java.util.HashSet<>();
        if (o instanceof IPrimitiveType<?> p) {
            Value x = el.get("value");
            if (x instanceof DateTime || x instanceof Date) FHIR.setPrimitiveValue(x, p);
            else if (x instanceof Time t) p.setValueAsString(t.toStringInner());
            else if (x instanceof org.opencds.cqf.cql.engine.runtime.String s) p.setValueAsString(s.getValue());
            else if (x instanceof org.opencds.cqf.cql.engine.runtime.Decimal d) p.setValueAsString(d.getValue().toPlainString());
            else if (x instanceof org.opencds.cqf.cql.engine.runtime.Boolean b) p.setValueAsString(java.lang.Boolean.toString(b.getValue()));
            else if (x instanceof org.opencds.cqf.cql.engine.runtime.Integer i) p.setValueAsString(java.lang.Integer.toString(i.getValue()));
            else if (x instanceof org.opencds.cqf.cql.engine.runtime.Long l) p.setValueAsString(java.lang.Long.toString(l.getValue()));
            else if (x != null) throw new EncodingError("primitive " + ci.getType().getLocalPart() + " value " + x.getClass().getName());
            rebuilt.add("value");
            for (Value item : items(el.get("extension"))) {
                if (!(item instanceof ClassInstance c) || !(toHapi(c) instanceof org.hl7.fhir.r4.model.Extension e)) {
                    throw new EncodingError("primitive " + ci.getType().getLocalPart() + " extension " + item.getClass().getName());
                }
                ((org.hl7.fhir.r4.model.Element) o).addExtension(e);
            }
            rebuilt.add("extension");
        } else {
            BaseRuntimeElementCompositeDefinition<?> def = (BaseRuntimeElementCompositeDefinition<?>)
                    FHIR.getFhirContext().getElementDefinition(((IBase) o).getClass());
            for (BaseRuntimeChildDefinition child : def.getChildren()) {
                rebuilt.add(child.getElementName());
                for (Value item : items(el.get(child.getElementName()))) {
                    if (!(item instanceof ClassInstance c)) throw new EncodingError(
                            ci.getType().getLocalPart() + "." + child.getElementName() + " " + item.getClass().getName());
                    BaseRuntimeElementDefinition<?> block = child.getChildByName(child.getElementName());
                    child.getMutator().addValue((IBase) o, block instanceof RuntimeResourceBlockDefinition b
                            ? toHapi(c, b.newInstance()) : toHapi(c));
                }
            }
        }
        if (o instanceof IBaseElement e && el.get("id") instanceof org.opencds.cqf.cql.engine.runtime.String id) {
            e.setId(id.getValue());
            rebuilt.add("id");
        }
        for (Map.Entry<String, Value> e : el.entrySet()) {
            if (e.getValue() != null && !rebuilt.contains(e.getKey())) {
                throw new EncodingError(ci.getType().getLocalPart() + "." + e.getKey() + " is not rebuilt");
            }
        }
        return (IBase) o;
    }

    /** The values of an element: none, one, or a list's. */
    static Iterable<Value> items(Value x) {
        if (x == null) return List.of();
        return x instanceof org.opencds.cqf.cql.engine.runtime.List l ? l : List.of(x);
    }
}
