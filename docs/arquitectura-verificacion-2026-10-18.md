# Verificación de go-cql: oráculos, regresión y deuda de diseño

Fecha: 18 de octubre de 2026. Código: `main` en `c3dbd21` más la rama
`fix/as-any` (`328f801`). Este documento continúa el
[análisis del 13 de octubre](analisis-proyecto-2026-10-13.md). Aquel fijó la
estrategia, y este registra qué se implementó de ella, qué faltó y qué
decisiones hay que tomar antes de seguir corrigiendo semántica.

## Dictamen

Las correcciones de los últimos PRs no convergían: `fix/definition-scope`
necesitó cinco rondas de review y `fix/as-any` cuatro, y en cada ronda
aparecían defectos nuevos. Medido ronda por ronda, la mayoría de esos defectos
los introducían los arreglos de la ronda anterior, en zonas que ninguna
verificación automática cubría. Hay dos causas, y ninguna es la falta de tests:

1. **Falta un oráculo para la capa FHIR.** El arnés ejecuta Java 3.26.0 y
   cql-execution solo sobre expresiones sin datos. `Evaluate.java` no tiene
   proveedor de datos, así que retrieve, choices, primitivos FHIR y FHIRHelpers
   no tienen oráculo externo. Las regresiones de `fix/as-any` estaban todas en
   esa capa.
2. **El diseño tiene deuda que obliga a parchear.** La identidad de tipo se
   decide comparando texto, Integer y Long comparten tipo en tiempo de
   ejecución, `toDecimal` interpreta como cero lo que no es número y la
   procedencia FHIR o System depende de una marca privada de la dependencia.
   Cada corrección local se filtra por esos puntos.

Recomendación: antes de corregir más semántica, primero la red de seguridad
(snapshots de respuestas y un oráculo con datos FHIR) y después la deuda de
diseño, un punto por PR.

## 1. Qué se implementó desde el 13 de octubre

| Propuesta del 13-10 | Estado |
|---|---|
| Ejecución Java con resultados tipados | Hecho: `TestRuntimeMatchesTheReference`, engine 3.26.0 (#120) |
| Segundo runtime JS | Hecho: cql-execution 3.3.2 (#122) |
| Generación de casos | Hecho, parcial: `gen` con las áreas `arithmetic`, `integer`, `scope` y `types` (#133 en adelante) |
| Comparador de valores independiente del motor | Parcial: `compare` lee las respuestas de ambos motores, con números por valor y listas |
| Registro de excepciones con respuesta exacta | Hecho para el arnés runtime: `decidedRuntimeAnswers`, `disputedRuntimeAnswers` y `notOracle` |
| `aggregate` con `with`/`without`, `distinct` por texto, duplicados de `union`/`except` | Corregidos (#123, #125, #127 y #128) |
| Oráculo con pacientes y bundles FHIR | **No hecho** |
| Manifiestos con conteos de casos obligatorios | **No hecho** |
| Rechazo estricto de ELM desconocido | **Abierto**, medido el 18-10: un nodo desconocido se importa como literal null |
| Contrato de cancelación | **Abierto**, medido: un contexto ya cancelado devuelve 42 sin error |
| Validación semántica de las librerías incluidas | **Abierto**, medido: una librería incluida con una rama inválida devuelve 1 |
| Documento completo del traductor oficial (`{"library": …}`) | **Abierto**, medido: `UnmarshalLibrary` devuelve una librería vacía sin error |

## 2. Lo que mostraron las últimas rondas

Esto es lo que se midió en los PRs #134, #135, #137 y en `fix/as-any`:

- **Java no sirve de oráculo en varias áreas.** Da la vuelta en el desborde
  de Integer, donde la especificación pide null; falla con `Any` como destino
  de un `as`, `is` o `cast`; trata cualquier lista como `List<String>`;
  `Product` da null en cuanto hay un elemento null; y NaN termina en error.
  En esas áreas hizo falta un oráculo propio: el de `integer` se escribió en
  Python, y vive fuera del repositorio.
- **La diferencial no veía lo que tocaban los arreglos.** Las áreas generadas
  usan literales de sistema. Los arreglos de `fix/as-any` eran sobre datos
  FHIR, y ahí la verificación volvió a ser a ojo.
- **Los tests confirmaban el comportamiento nuevo en vez de buscar
  regresiones.** Una regresión que hacía fallar la aritmética temporal de toda
  fecha FHIR sin hora pasó con `go test ./...` en verde. Dos tests pasaban igual
  en `main`, así que no protegían nada.
- **La diferencial `main` contra rama, aplicada en la última ronda,
  convergió.** Sobre 14.040 expresiones cambiaron 1.447 respuestas, y todas se
  pudieron clasificar como cambios buscados. Ese método detectó dos
  regresiones que ningún test cubría. No necesita oráculo, y es la base de la
  decisión D2.

## 3. Deuda de diseño

Cada punto se midió en esta sesión. Están ordenados por cuántos arreglos
bloquean.

1. **Identidad de tipo por texto.** `Value.Type()` devuelve un nombre, y
   `isTemporalType`, `valueIsOf` y el despacho aritmético lo comparan. Cuando
   `asDateTime` le asignó el tipo FHIR `"dateTime"` a un valor para conservar
   su procedencia, la aritmética temporal dejó de reconocerlo. Un valor no
   puede llevar su modelo sin cambiar su identidad.
2. **Nombres de tipo sin espacio de nombres.** Como consecuencia del punto 1,
   `{ 1, 2 } is FHIR.Resource` daba true, porque una lista de CQL se llama
   `"List"`, igual que el recurso List de FHIR. `FHIR.code { … }` se materializa
   como un Code de sistema porque `EqualFold("code", "Code")`, y
   `FHIR.Quantity { … }` como un Quantity de sistema, perdiendo sus valores.
3. **Procedencia FHIR o System.** Depende de `read`, un campo privado de
   fhirpath que este repositorio no puede fijar, así que se pierde al
   transformar un valor: una fecha FHIR reescrita como DateTime deja de ser
   FHIR.
4. **Integer y Long son el mismo tipo en tiempo de ejecución**
   (`fptypes.Integer`, int64). El límite de 32 bits se dedujo del tipo
   estático, con una lista de «Integers certeros» que hay que mantener a mano;
   `5L is Integer` da true.
5. **`toDecimal` interpreta como cero lo que no es número.** Causó los 0
   silenciosos de la negación, `Abs`, `Sum`, `Product` y `Avg`. Se cerró
   operador por operador, y el lector sigue intacto.
6. **Sema tipa `Unknown op T` como `T`.** Por eso el tipo estático no es
   confiable cuando un operando viene de una librería incluida o de un alias.

## 4. Decisiones

Cada una se toma o se rechaza antes de su PR.

**D1. Oráculo con datos FHIR.** Extender `Evaluate.java` con un proveedor
sobre bundles fijados y el FHIRHelpers oficial, y la parte de JS con
cql-exec-fhir. Los bundles viven en `testdata/reference/fhir/`, sintéticos y
sin datos de pacientes reales. Es lo de mayor retorno, porque cubre la capa
donde más se parcheó.

**D2. Snapshots de respuestas versionados.** Cada área (`arithmetic`,
`integer`, `scope`, `types` y una nueva `fhir`) guarda su corpus y la respuesta
del motor a cada expresión en `testdata/snapshots/`. Un test compara cada
respuesta y falla si alguna cambia; un comando regenera el snapshot, y el diff
del PR muestra exactamente qué respuestas cambian. Así cada PR justifica sus
cambios de comportamiento uno por uno, con o sin oráculo. Reemplaza la
diferencial ad hoc contra `main`.

**D3. Política de oráculos explícita.** Mandan la especificación y la
conformance. Java es la señal principal, cql-execution la segunda, y nunca se
decide por mayoría. Cada área declara en un manifiesto qué oráculo usa, y cada
excepción registra la respuesta exacta y su razón, como ya hace el arnés
runtime. El oráculo de especificación del área `integer` pasa de Python a Go,
dentro de `compare`.

**D4. La deuda de diseño se ataca en la raíz, un punto por PR**, con su
snapshot y su diferencial antes de tocar código:

- (a) **Identidad de tipo:** un descriptor `{modelo, nombre}` en vez del texto
  de `Type()`. Desbloquea los puntos 2 y 3 de la sección 3.
- (b) **Un solo lector numérico,** que da error ante lo que no es número.
- (c) **Un tipo Long propio** en tiempo de ejecución.
- (d) **Una marca en sema** para los tipos inferidos desde Unknown o Any.

**D5. Proceso.** Un hallazgo de review se arregla en el punto exacto que
señala, y lo que lo excede va a otro PR. Cada test nuevo debe fallar en
`main`. El snapshot y la diferencial se corren antes del review, no después.

## 5. Plan por PRs

| PR | Contenido | Criterio de aceptación |
|---|---|---|
| 1 | **Snapshots (D2).** Corpus y respuestas de las cuatro áreas actuales, el test de comparación y el comando de regeneración | Cambiar una respuesta del motor hace fallar el test; regenerar produce un diff revisable; los manifiestos fijan el número de expresiones |
| 2 | **Oráculo FHIR (D1).** Bundles sintéticos, `Evaluate.java` y `evaluate.js` con datos, área `fhir` en `gen` y en el snapshot | Las filas sobre datos FHIR se comparan contra Java y cql-execution; si falta un artefacto obligatorio, la prueba falla en vez de saltarse |
| 3 | **Oráculo propio en Go y manifiesto de oráculos (D3)** | `compare -oracle spec` reproduce el resultado del oráculo Python sobre las semillas medidas |
| 4 | **Contratos abiertos del 13-10:** ELM desconocido, cancelación, validación de includes y documento del traductor | Cada caso medido el 18-10 tiene su test, que falla en `main` |
| 5 a 8 | **Deuda D4 (a), (b), (c) y (d)**, en ese orden | Snapshot sin cambios fuera de lo declarado en el PR; diferencial contra Java sin diferencias nuevas |

Los PRs 1 y 2 no cambian la semántica del motor, y van antes que cualquier
otra corrección. Los pendientes anotados en estas semanas (llamadas fluidas a
librerías incluidas, `Extension.value[x]`, `cast` que convierte, elementos FHIR
repetidos, materialización de instancias FHIR) se retoman después del PR 5,
porque casi todos dependen de la identidad de tipo.

## 6. Preguntas abiertas

- **Versión de FHIRHelpers para el oráculo:** la que trae el motor, o la
  publicada junto con el traductor 3.26.0.
- **Forma de la respuesta en el snapshot:** el texto actual de `compare`, o un
  formato tipado como el que propuso el análisis del 13-10 (`kind`, `value` y
  estado `value`/`error`/`unsupported`). El tipado cuesta más, pero sirve
  también como normalizador.
- **CI:** si los snapshots corren en cada PR, como `go test`, y Java y JS en un
  job nocturno con imágenes fijadas, como propuso el 13-10.
