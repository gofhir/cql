import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.SerializationFeature;
import ca.uhn.fhir.context.FhirContext;
import ca.uhn.fhir.parser.IParser;
import com.fasterxml.jackson.databind.JsonNode;
import org.apache.commons.lang3.tuple.Pair;
import org.cqframework.cql.cql2elm.*;
import org.hl7.elm.r1.ExpressionDef;
import org.hl7.elm.r1.FunctionDef;
import org.hl7.elm.r1.Library;
import org.hl7.elm.r1.VersionedIdentifier;
import org.hl7.fhir.instance.model.api.IBaseResource;
import org.hl7.fhir.r4.model.Base;
import org.hl7.fhir.r4.model.PrimitiveType;
import org.opencds.cqf.cql.engine.data.CompositeDataProvider;
import org.opencds.cqf.cql.engine.data.DataProvider;
import org.opencds.cqf.cql.engine.execution.CqlEngine;
import org.opencds.cqf.cql.engine.execution.Environment;
import org.opencds.cqf.cql.engine.fhir.model.R4FhirModelResolver;
import org.opencds.cqf.cql.engine.retrieve.RetrieveProvider;
import org.opencds.cqf.cql.engine.runtime.*;
import java.io.ByteArrayInputStream;
import java.io.InputStream;
import java.math.BigDecimal;
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
import java.util.Set;
import java.util.TreeMap;

/** Ejecuta cada definición con el engine de referencia y registra el valor
 *  tipado, para que lo que el motor responde se compare contra un resultado
 *  y no contra lo que uno cree que la spec dice.
 *
 *  La salida no pasa por nada de go-cql: un oráculo que usara su String() o su
 *  igualdad heredaría los mismos defectos que tiene que encontrar. */
public class Evaluate {
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
        libs.getLibrarySourceLoader().registerProvider(new DefaultLibrarySourceProvider(dir));
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
                entry = encode(engine.evaluate(lib.getIdentifier(), Set.of(def.getName()), null, null, null, EVALUATION)
                        .forExpression(def.getName()).value());
            } catch (Exception | StackOverflowError e) {
                // Un error es una respuesta: se registra, no aborta el resto. Una
                // recursión sin fondo también, aunque Java no la cuente como Exception.
                Map<String, Object> err = new LinkedHashMap<>();
                err.put("error", e.getClass().getSimpleName());
                err.put("message", String.valueOf(e.getMessage()));
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
        out.put("engine", CqlEngine.class.getPackage().getImplementationVersion());
        // ISO completo: toString() omite los segundos cuando son cero.
        out.put("evaluationDateTime", EVALUATION.format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME));
        out.put("results", results);
        System.out.println(new ObjectMapper().enable(SerializationFeature.INDENT_OUTPUT).writeValueAsString(out));
    }

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
            public InputStream getLibrarySource(VersionedIdentifier id) {
                String src = sources.get(id.getId());
                return src == null ? null : new ByteArrayInputStream(src.getBytes(StandardCharsets.UTF_8));
            }
        });
        libs.getLibrarySourceLoader().registerProvider(new DefaultLibrarySourceProvider(Paths.get(args[1])));
        libs.getCqlCompilerOptions().setOptions(
                CqlCompilerOptions.Options.EnableResultTypes,
                CqlCompilerOptions.Options.EnableLocators);

        Map<String, DataProvider> data = new HashMap<>();
        Pair<String, Object> context = null;
        // "-" for datos.json evaluates without data, so that elm-salida.json can
        // still be asked for.
        if (args.length > 2 && !args[2].equals("-")) {
            FhirBundle bundle = FhirBundle.read(Paths.get(args[2]));
            data.put("http://hl7.org/fhir", new CompositeDataProvider(new R4FhirModelResolver(), bundle));
            context = Pair.of("Patient", bundle.patientId);
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
                err.put("message", String.valueOf(t.getErrors().get(0).getMessage()));
                results.put(name, err);
                continue;
            }
            Library lib = t.toELM();
            elms.put(name, new ObjectMapper().readTree(t.toJson()));
            Object entry;
            try {
                entry = encode(engine.evaluate(lib.getIdentifier(), Set.of(name), context, null, null, EVALUATION)
                        .forExpression(name).value());
            } catch (Exception | StackOverflowError e) {
                Map<String, Object> err = new LinkedHashMap<>();
                err.put("error", e.getClass().getSimpleName());
                err.put("message", String.valueOf(e.getMessage()));
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
        out.put("engine", CqlEngine.class.getPackage().getImplementationVersion());
        out.put("evaluationDateTime", EVALUATION.format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME));
        out.put("results", results);
        System.out.println(new ObjectMapper().enable(SerializationFeature.INDENT_OUTPUT).writeValueAsString(out));
    }

    /** Los recursos de datos.json, servidos por tipo. Sin filtros de código ni de
     *  fecha: un retrieve devuelve todos los de su tipo, y es la expresión la que
     *  filtra, como en el proveedor del snapshot de go-cql. */
    static final class FhirBundle implements RetrieveProvider {
        final Map<String, List<Object>> byType = new HashMap<>();
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
                    b.byType.computeIfAbsent(e.getKey(), k -> new ArrayList<>()).add(r);
                    if (e.getKey().equals("Patient")) b.patientId = r.getIdElement().getIdPart();
                }
            });
            return b;
        }

        @Override
        public Iterable<Object> retrieve(String context, String contextPath, Object contextValue, String dataType,
                String templateId, String codePath, Iterable<Code> codes, String valueSet, String datePath,
                String dateLowPath, String dateHighPath, Interval dateRange) {
            return byType.getOrDefault(dataType, List.of());
        }
    }

    /** Codifica un valor con su tipo CQL explícito. null es JSON null; una lista
     *  vacía sigue siendo una lista, para no confundir las dos. Los números van
     *  como texto, para que 1 y 1.0 no se vuelvan iguales al serializar. */
    static Object encode(Object v) {
        if (v == null) return null;
        Map<String, Object> m = new LinkedHashMap<>();
        if (v instanceof Boolean b) {
            m.put("type", "Boolean"); m.put("value", b);
        } else if (v instanceof Integer || v instanceof Long) {
            m.put("type", v instanceof Integer ? "Integer" : "Long"); m.put("value", v.toString());
        } else if (v instanceof BigDecimal d) {
            m.put("type", "Decimal"); m.put("value", d.toPlainString());
        } else if (v instanceof String s) {
            m.put("type", "String"); m.put("value", s);
        } else if (v instanceof Quantity q) {
            m.put("type", "Quantity"); m.put("value", q.getValue().toPlainString()); m.put("unit", q.getUnit());
        } else if (v instanceof DateTime d) {
            // toString() es para leerlo: omite el offset en unas precisiones y lo
            // agrega en otras. Lo que se compara es instant, ISO completo siempre.
            m.put("type", "DateTime"); m.put("value", d.toString()); m.put("precision", d.getPrecision().toString());
            m.put("offset", String.valueOf(d.getZoneOffset()));
            m.put("instant", d.getDateTime().format(java.time.format.DateTimeFormatter.ISO_OFFSET_DATE_TIME));
        } else if (v instanceof Date d) {
            m.put("type", "Date"); m.put("value", d.toString()); m.put("precision", d.getPrecision().toString());
        } else if (v instanceof Time d) {
            m.put("type", "Time"); m.put("value", d.toString()); m.put("precision", d.getPrecision().toString());
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
        } else if (v instanceof PrimitiveType<?> p) {
            // Un primitivo FHIR, no uno de System: su tipo es el del modelo
            // (FHIR.dateTime), y su valor el texto que trae el recurso.
            m.put("type", "FHIR." + p.fhirType()); m.put("value", p.getValueAsString());
        } else if (v instanceof Base b) {
            // Un recurso o un elemento FHIR: su tipo del modelo y su JSON.
            m.put("type", "FHIR." + b.fhirType());
            m.put("value", b instanceof IBaseResource r
                    ? FhirContext.forR4Cached().newJsonParser().encodeResourceToString(r)
                    : FhirContext.forR4Cached().newJsonParser().encodeToString((org.hl7.fhir.instance.model.api.IBase) b));
        } else if (v instanceof Iterable<?> it) {
            List<Object> el = new ArrayList<>();
            for (Object x : it) el.add(encode(x));
            m.put("type", "List"); m.put("elements", el);
        } else {
            // Sin codificación propia: se deja visible en vez de adivinarla.
            m.put("type", "Unencoded:" + v.getClass().getName()); m.put("value", v.toString());
        }
        return m;
    }
}
