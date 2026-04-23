#!/usr/bin/env bash

set -eu pipefail

encode() {
  python3 -c "import urllib.parse, sys; print(urllib.parse.quote(sys.argv[1]))" "$1"
}

username() {
  python3 -c "import sys; m = sys.argv[1].split('.')[1].split('@')[0] if sys.argv[1].startswith('gustav') else sys.argv[1].split('.')[0]; print(m[0].upper() + m[1:])" "$1"
}

echo "username,password_hash,grp" >> for_database.csv

echo "<!doctype html><html><body>" >> mailing_links.html

while read -r email
do
  uname="$(username "$email")"
  pw="$(pwgen -v 20 1)"
  hash="\"$(authelia crypto hash generate --password "$pw" | cut -d' ' -f 2)\""
  subject="Dina inloggningsuppgifter till klassens.spetsen.net"
  encoded_subject="$(encode "$subject")"
  grp_uid="ff7c1374-4779-45ca-9758-35475153f27d"
  body="Hej $uname,

Nedan följer dina inloggningsuppgifter till https://klassens.spetsen.net.

Regler för tjänsten och klassensomröstningen är:
1. Vem som föreslagit något visas, så föreslå inget du inte hade sagt till en viss person fysiskt.
2. Det är inte tillåtet att utnyttja tekniska sårbarheter i plattformen för att till exempel manipulera omröstningen. Om du hittar en sårbarhet, vänligen kontakta Eric.
3. Denna tjänst är till för att man ska kunna ha lång betänketid och kunna skriva förslag hela tiden, men det slutgiltiga beslutet kommer att tas av den som faktiskt gör tallrikarna utifrån resultatet i omröstningen.
4. Andreas kommer att granska alla förslag och motiveringar.

Reglerna kommer att finnas tillgängliga på https://klassens.spetsen.net/static/rules-230S.txt.

Håll dina inloggningsuppgifter hemliga, till exempel genom att spara dem i en lösenordshanterare.
Radera detta mejl när du sparat lösenordet på ett säkert ställe.

Om du vill byta lösenord eller förlorar det, kontakta Eric.

Användarnamn: $uname
Lösenord: $pw


Med vänlig hälsning,
Eric"
  encoded_body="$(encode "$body")"

  echo "<a href=\"mailto:${email}?subject=${encoded_subject}&body=${encoded_body}\">$email</a> <br>" >> mailing_links.html
  echo "$uname,$hash,$grp_uid" >> for_database.csv

done

echo "</body></html>" >> mailing_links.html
