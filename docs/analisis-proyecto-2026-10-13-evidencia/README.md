# Evidencia reproducible

Todos los inputs son sintéticos. Los archivos se añadieron para documentar el análisis; no cambian el runtime ni la suite del proyecto.

- `sondas.go.txt`: programa Go de sondas; extensión `.txt` para evitar crear un paquete adicional al ejecutar `go test ./...`.
- `probe.cql`: cinco definiciones aceptadas por el traductor Java 3.26.0.
- `probe.elm.json`: salida de ese traductor para el fixture. No contiene resultados del runtime Java.

Desde la raíz del repositorio:

```sh
cp docs/analisis-proyecto-2026-10-13-evidencia/sondas.go.txt /tmp/go-cql-review-probes.go
cp docs/analisis-proyecto-2026-10-13-evidencia/probe.elm.json /tmp/go-cql-review-reference-elm.json
go run /tmp/go-cql-review-probes.go
```

Resultados observados en el commit revisado:

| Sonda | Resultado observado | Esperado o alcance |
|---|---|---|
| `with` + aggregate | 6, sin error | 2 |
| `without` + aggregate | 6, sin error | 4 |
| `where X = 2` + aggregate | 2, sin error | Control correcto |
| `{1,1,2} union {2}` | `{1,1,2}` | Una aparición de cada elemento |
| `{1,1,2} except {3}` | `{1,1,2}` | Una aparición de cada elemento |
| distinct sobre `'nil'` y null | 1 | 2 |
| distinct sobre 1 y 1.0 | 2 | No clasificado como defecto; requiere fijar promoción/igualdad |
| distinct sobre cuatro parejas de strings | 3 | 4 |
| Contexto cancelado + literal 42 | 42, sin error | Contrato de cancelación por corregir/explicitar |
| Nodo ELM desconocido | Literal null; evaluación sin error | Rechazo explícito |
| Include con aritmética inválida ejecutada | Error runtime | No afirmar éxito ni resultado aritmético incorrecto |
| Include con rama inválida no ejecutada | 1, sin error | Rechazo semántico del grafo recomendado |
| Mismo include comprobado directamente | Diagnóstico semántico | Confirma diferencia entre raíz y dependencia |
| Documento ELM oficial completo | Biblioteca sin identifier/statements, sin error | Adaptar envelope y validar estructura |

Para regenerar ELM, usar las instrucciones de `internal/referenceharness/README.md` con `probe.cql` de esta carpeta. La generación realizada durante la revisión usó los jars y la clase ya presentes localmente:

```sh
java -cp 'internal/referenceharness/out:internal/referenceharness/lib/*' \
  Translate docs/analisis-proyecto-2026-10-13-evidencia/probe.cql \
  docs/analisis-proyecto-2026-10-13-evidencia \
  > /tmp/go-cql-review-reference-elm.json
```

La clase y los jars están ignorados por Git y pueden no existir en otro checkout. No usar su presencia como requisito implícito de CI.

Validación realizada adicionalmente:

```sh
go test ./... -coverprofile=/tmp/go-cql-review-coverage.out
go test -race ./...
go test ./... -coverpkg=./... -coverprofile=/tmp/go-cql-review-cross-coverage.out
go test -run '^$' -bench BenchmarkEvaluate -benchmem -benchtime=1s -count=3 .
```

Los tests sobre el checkout externo eCQM se omitieron al no disponer de `ECQM_CONTENT_DIR`. No se instalaron ni ejecutaron runtimes externos Java/JS/Google para comparar resultados o rendimiento; la propuesta de integración figura en el informe principal.
