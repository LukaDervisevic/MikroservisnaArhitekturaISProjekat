## MikroservisnaArhitekturaISProjekat
Projekat za predmet Mikroservisna arhitektura informacionih sistema na fakultetu organizacionih nauka  
Projekat je zasnovan na mirkoservisnoj arhitekturi, pocetna faza je kreiranje monolitnog projekta i razdvajanje na servise

# Serverske tehnologije:
1. Go
2. GORM objekto relacioni maper
3. GRPC za komenkciju izmedju mikroservisa
4. MigrateDB za migriranje relacionih baza
5. PostgreSQL kao relacioni SUBP
6. Docker - kontejnerizacija servisa
7. Github Actions - kreiranje CI/CD pipeline-ova

# Klijentske tehnologije:
1. TypeScript
2. React
3. ESLint
4. TailwindCSS

  

# Slanje mejlova preko reda poruka
Mejlovi se ne šalju direktno iz zahteva nego preko RabbitMQ reda `mail.send`:

1. **Producer — `lecture-service`**: pri svakom uspešnom kreiranju predavanja
   (`CreateLectureHandler`) stavlja zahtev (primalac, subject, body, timestamp)
   u red. Primalac je predefinisani mejl predavača (`lecturers.email`).
2. **Consumer — `lecturer-service`**: poseban worker (`internal/service/mail`)
   prazni red i svaku poruku stvarno šalje preko SMTP releja
   (`SMTPSender`, `net/smtp`, podržan `none` / `starttls` / `tls`).
3. **Rate limit**: sliding window (`internal/service/ratelimit`) — najviše
   `MAIL_RATE_LIMIT` poruka u bilo kom intervalu od `MAIL_RATE_PERIOD`
   (podrazumevano 10 / 1m). Tempo se računa iz stvarne istorije slanja, nema
   fiksnog `sleep`-a. Poruke koje trajno padaju odlaze u `mail.dlq` posle
   `MAIL_MAX_RETRIES` pokušaja.

Slanje ide preko **Gmail SMTP-a** (`smtp.gmail.com:587`, STARTTLS). U
`lecturer-service/.env.*` treba popuniti samo `MAIL_SMTP_USERNAME` (puna Gmail
adresa) i `MAIL_SMTP_PASSWORD` (app password iz
https://myaccount.google.com/apppasswords, uz uključenu 2FA); `MAIL_FROM_ADDRESS`
podrazumevano preuzima vrednost username-a jer Gmail prihvata samo svoju adresu
kao pošiljaoca. Za drugi relej dovoljno je promeniti `MAIL_SMTP_HOST/PORT/TLS`
(`none` / `starttls` / `tls`).

Testovi: `go test ./lecturer-service/internal/service/...` — simulacija 25
mejlova odjednom uz proveru da nijedan minut ne sadrži više od 10 slanja, plus
SMTP test protiv lažnog releja u samom testu.
