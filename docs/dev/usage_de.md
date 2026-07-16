# Verwendung

Um ein Support-Archiv zu erstellen, wenden Sie die [crd lib](https://github.com/cloudogu/k8s-support-archive-lib), den [Operator](https://github.com/cloudogu/k8s-support-archive-operator) und eine benutzerdefinierte Ressource für das Support-Archiv im Cluster an.
Informationen zum Format der benutzerdefinierten Ressource finden Sie in der [CRD-Bibliothek](https://github.com/cloudogu/k8s-support-archive-lib/blob/develop/k8s/helm-crd/templates/k8s.cloudogu.com_supportarchives.yaml).

Das Deployment des Operators enthält einen Nginx-Sidecar-Container mit einem gemeinsam genutzten Volume, um das erstellte Support-Archiv bereitzustellen.

## Interne Prozesse

### Finalizer

Der ausgelöste Reconciler prüft zunächst, ob ein Finalizer vorhanden ist, und fügt einen hinzu, falls noch keiner definiert ist.
Mithilfe des Finalizers kann der Betreiber später die Support-Archive löschen, wenn die CR gelöscht wird.

### Reconciler

Im Allgemeinen versucht der Reconciler stets, die CR erneut in die Queue einzureihen, um Blockierungen zu vermeiden.  
Dies geschieht z.B. nachdem ein Finalizer hinzugefügt wurde oder nachdem ein einzelner Collector zur Datenerhebung ausgeführt wurde.

### Zustand

Beim Erstellen des Support-Archivs prüft der Operator zunächst die Metadaten des aktuellen Zustands des Archivs.  
Jeder Collector-Typ besitzt eine eigene Zustandsdatei `.done` in jedem Archiv unter `/data/work/<namespace/<name>/<type>`, die vom Nginx-Sidecar aus nicht zugänglich ist.  
Die Existenz dieser Datei zeigt an, dass der Collector die Daten erfolgreich abgerufen hat.

Der Operator speichert den Zustand (das resultierende Archiv) als `ZIP` unter folgendem Pfad: `/data/supportarchives/namespace/name`.  
Um Speicherüberläufe zu vermeiden, wird empfohlen, einen gepufferten Stream zu verwenden.

### Collectors

Collectors sind dafür verantwortlich, einzelne Datenbereiche für das Archiv zu sammeln, z.B. Logs, Kubernetes-Ressourcen oder Health-Informationen.  
Eine Liste der Collector-Typen definiert den Umfang bzw. die Vollständigkeit eines Support-Archivs.