package mail

import (
	"fmt"
	"strings"
	"time"

	"github.com/LukaDervisevic/MikroservisnaArhitekturaISProjekat/lecture-service/internal/model"
)

func LectureCreated(lecture *model.Lecture) (subject, body string) {
	subject = fmt.Sprintf("Zakazano novo predavanje: %s", lecture.Name)

	var sb strings.Builder
	if lecture.Lecturer != nil {
		fmt.Fprintf(&sb, "Zdravo %s,\n\n", lecture.Lecturer.FullName)
	} else {
		sb.WriteString("Zdravo,\n\n")
	}
	fmt.Fprintf(&sb, "novo predavanje Vam je zakazano\n\n")
	fmt.Fprintf(&sb, "Predavanje: %s\n", lecture.Name)
	fmt.Fprintf(&sb, "Trajanje: %d minuta\n", lecture.Duration)

	if lecture.Event != nil {
		fmt.Fprintf(&sb, "Dogadjaj: %s (%s)\n", lecture.Event.Name, lecture.Event.Type)
		if lecture.Event.DateTime > 0 {
			fmt.Fprintf(&sb, "Vreme početka: %s\n", time.Unix(lecture.Event.DateTime, 0).UTC().Format(time.RFC1123))
		}
		if lecture.Event.Location != nil {
			fmt.Fprintf(&sb, "Lokacija: %s, %s\n", lecture.Event.Location.Name, lecture.Event.Location.Address)
		}
	}

	return subject, sb.String()
}
