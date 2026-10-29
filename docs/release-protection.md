# Compilacion y firma de las versiones instalables

Desde 1.2.3, por instruccion del titular tras el rechazo de seguridad de Store,
Aura se compila con Go estandar, sin Garble, sin ofuscacion de literales y sin
modo tiny. El script ya no instala ni ejecuta el ofuscador y no crea semillas.
Se conserva la informacion de compilacion, simbolos y diagnosticos normales de
Go. -trimpath elimina rutas del entorno de compilacion; no ofusca nombres ni
literales. -H=windowsgui selecciona el subsistema grafico.

Los paquetes antiguos 1.2.1 y 1.2.2 permanecen como evidencia y no se sobrescriben.
Sus semillas anteriores siguen en el directorio privado ignorado por Git; no
se utilizan en la nueva compilacion ni se distribuyen.

El rechazo de Microsoft identifica aura-native.exe del MSIX 1.2.1.0 como
detectado por su escaner. No proporciona el nombre de deteccion ni prueba que
Garble sea la causa. Quitar la ofuscacion facilita la inspeccion y el diagnostico,
pero no acredita por si solo ausencia de malware ni aprobacion de Microsoft.

La licencia del codigo original sigue siendo privativa. No se incluyen fuentes
originales en el paquete; quitar la ofuscacion no equivale a publicar el codigo.
Las dependencias conservan sus licencias y sus derechos de reemplazo aplicables.

Los instaladores EXE siguen sin Authenticode. Microsoft firma los MSIX aprobados
para distribucion en Store; la firma no sustituye el escaneo antimalware.
Para revisar una deteccion: https://www.microsoft.com/en-us/wdsi/filesubmission
No se han enviado binarios automaticamente a ese servicio ni a otros terceros.
