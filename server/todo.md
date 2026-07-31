# Server TODO

Der Server ist fuer einen kleinen Single-Node-Betrieb grundsaetzlich valide. Vor einem groesseren Produktiveinsatz bleiben folgende Punkte offen:

## Prioritaet 1

- [ ] Account- und Device-Quotas atomar innerhalb der jeweiligen Registrierungstransaktion pruefen und durchsetzen.
- [ ] Parallelregistrierungen mit Race- und HTTP-Integrationstests abdecken.

## Prioritaet 2

- [ ] Pull-Pagination und eine maximale Seitengroesse einfuehren.
- [ ] JWT-Secret-Rotation mit einer kontrollierten Uebergangsphase ermoeglichen.
- [ ] Strukturierte Logs, Metriken und Alarmierung fuer Fehler, Konflikte, Limits und Wartungslaeufe ergaenzen.
- [ ] Externes SQLite-Backup- und Restore-Verfahren fuer den Produktivbetrieb dokumentieren und testen.

## Spaeter / bei mehreren Serverinstanzen

- [ ] Pairing-Challenges aus dem Prozessspeicher in einen gemeinsam nutzbaren, ablaufenden Speicher verschieben.
- [ ] IP- und Account-Rate-Limits instanzuebergreifend speichern und durchsetzen.

## Deployment-Checkliste

- [ ] Backend nur an `127.0.0.1` oder ein geschuetztes internes Netz binden.
- [ ] Nur bekannte Reverse-Proxies in `trusted_proxy_cidrs` eintragen.
- [ ] Starkes, individuelles JWT-Secret verwenden und Config/DB-Dateirechte beschraenken.
- [ ] Apache-Weiterleitung und `X-Forwarded-Proto: https` im Zielsystem pruefen.
