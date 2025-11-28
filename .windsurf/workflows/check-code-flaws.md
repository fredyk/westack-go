---
description: Busca debilidades en el código
---

## INPUT

Ninguno, este es un workflow autónomo

## CONTEXT

Eres el mejor programador Go del mundo, con especial experiencia en diseño de frameworks
para APIs REST.

## INSTRUCTIONS

- westack-go es un framework extremadamente grande, por lo que nunca vas a tener una visión
absolutamente global de el por limitaciones de ventana de contexto.
- Para suplir este problema, cada vez que se invoque este workflow, "aprenderás un poquito más de westack-go" leyendo trocitos de código "aquí y allá".
- Lo que vayas aprendiendo lo incorporarás a memorias internas de Cascade, y además también al
lugar adecuado en la documentación bajo docs/<mm-some-category>/<nn-some-subcategory>/<pp-some-file.md>
- Cuando tengas suficiente contexto global de un aspecto de westack-go, utilizarás ese aprendizaje para auditar algún pequeño fragmento de código.
- Si encuentras algo que pueda ser un problema, lo marcarás con un warning en el propio código, prefijado con algo así:
`// TODO: <mm-some-category>/<nn-some-subcategory>/<pp-some-file.md> <mm-some-problem>`
- La herramienta de `sequentialthinking` te ayudará mucho en estas tareas complejas de audiotoría. Tú mismo irás aprendiendo con tus memorias internas de Cascade cuántos pasos de sequentialthinking son adecuados para cada tipo de auditoría.
- Cuando en una auditoría encuentres un fallo extremadamente grave que tenga que ver con seguridad o permisos (RBAC, tokens, etc.), en esos casos aplicarás directamente los fixes en el código, pero siempre tendrás que estar muy muy seguro de que el fix es correcto.
- Todo fix irá acompañado siempre de invocar el script `cd v2 && ./run_tests.sh` (quizás tengas que levantar un Mongo con docker primero) y deben pasar TODOS los tests sin hacer trampa.
- La documentación nunca contendrá código interno de implementación, como mucho ejemplos de uso de interfaces.
- La documentación siempre debe quedar bien estructurada obligatoriamente en `docs/<mm-some-category>/<nn-some-subcategory>/<pp-some-file.md>` ya que es demasiado grande y debe ser sencillo para los devs moverse entre diferentes secciones. También debe haber suficientes enlaces internos entre los .md
- JAMÁS JAMÁS JAMÁS puedes eliminar tests anteriores, e incluso tienes prácticamente prohibido modificarlos. Si están ahí es porque un día fueron útiles para hacer el código robusto.

### NOTAS

- Es MUY MUY MUY importante que la documentación que crees o las memorias internas que añadas
sean 100% fieles a la implementación. El codebase es tan grande que probablemente un pequeño fragmento de él aparentemente inofensivo podría tener implicaciones en otro punto muy lejano
del código que inicialmente no parecía estar relacionado. 

## OUTPUT

Un breve resumen de las correcciones aplicadas, marcando con warnings aquello que podría afectar
a la funcionalidad.