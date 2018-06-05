package main

import (
	"net/http"
)

func domainStatistics(domain string) ([]int64, error) {
	statement := `
		SELECT COUNT(views.viewDate)
		FROM (
			SELECT to_char(date_trunc('day', (current_date - offs)), 'YYYY-MM-DD') AS date
			FROM generate_series(0, 30, 1) AS offs
		) gen LEFT OUTER JOIN views
		ON gen.date = to_char(date_trunc('day', views.viewDate), 'YYYY-MM-DD') AND
		   views.domain=$1
		GROUP BY gen.date
		ORDER BY gen.date;
	`
	rows, err := db.Query(statement, domain)
	if err != nil {
		logger.Errorf("cannot get daily views: %v", err)
		return []int64{}, errorInternal
	}

	defer rows.Close()

	last30Days := []int64{}
	for rows.Next() {
		var count int64
		if err = rows.Scan(&count); err != nil {
			logger.Errorf("cannot get daily views for the last month: %v", err)
			return []int64{}, errorInternal
		}
		last30Days = append(last30Days, count)
	}

	return last30Days, nil
}

func domainStatisticsHandler(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Session *string `json:"session"`
		Domain  *string `json:"domain"`
	}

	var x request
	if err := unmarshalBody(r, &x); err != nil {
		writeBody(w, response{"success": false, "message": err.Error()})
		return
	}

	o, err := ownerGetBySession(*x.Session)
	if err != nil {
		writeBody(w, response{"success": false, "message": err.Error()})
		return
	}

	domain := stripDomain(*x.Domain)
	isOwner, err := domainOwnershipVerify(o.OwnerHex, domain)
	if err != nil {
		writeBody(w, response{"success": false, "message": err.Error()})
		return
	}

	if !isOwner {
		writeBody(w, response{"success": false, "message": errorNotAuthorised.Error()})
		return
	}

	viewsLast30Days, err := domainStatistics(domain)
	if err != nil {
		writeBody(w, response{"success": false, "message": err.Error()})
		return
	}

	commentsLast30Days, err := commentStatistics(domain)
	if err != nil {
		writeBody(w, response{"success": false, "message": err.Error()})
		return
	}

	writeBody(w, response{"success": true, "viewsLast30Days": viewsLast30Days, "commentsLast30Days": commentsLast30Days})
}