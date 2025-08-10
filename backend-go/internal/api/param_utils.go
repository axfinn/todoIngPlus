package api

import (
    "net/url"
    "strconv"
    "strings"
)

// ParsedUpcomingParams 结构化解析结果
type ParsedUpcomingParams struct {
    Hours   int
    Limit   int
    Sources []string
}

// parseUpcomingParams 解析 hours/limit/sources 并做边界校验
func parseUpcomingParams(q url.Values) ParsedUpcomingParams {
    hours := 24 * 7
    if h := q.Get("hours"); h != "" {
        if v, err := strconv.Atoi(h); err == nil && v > 0 && v <= 24*30 { hours = v }
    }
    limit := 0
    if l := q.Get("limit"); l != "" { if v, err := strconv.Atoi(l); err == nil && v>0 && v<=500 { limit = v } }
    var sources []string
    if s := q.Get("sources"); s != "" {
        raw := strings.Split(s, ",")
        for _, r := range raw {
            t := strings.TrimSpace(r)
            if t == "" { continue }
            if t == "task" || t == "event" || t == "reminder" { sources = append(sources, t) }
        }
    }
    return ParsedUpcomingParams{Hours: hours, Limit: limit, Sources: sources}
}

// ParsedCalendarParams 结构化解析结果
type ParsedCalendarParams struct {
    Year    int
    Month   int
    Limit   int
    Sources []string
}

func parseCalendarParams(q url.Values, nowYear int, nowMonth int) ParsedCalendarParams {
    year := nowYear
    month := nowMonth
    if y := q.Get("year"); y != "" { if v, err := strconv.Atoi(y); err == nil { year = v } }
    if m := q.Get("month"); m != "" { if v, err := strconv.Atoi(m); err == nil && v>=1 && v<=12 { month = v } }
    limit := 0
    if l := q.Get("limit"); l != "" { if v, err := strconv.Atoi(l); err == nil && v>0 && v<=1000 { limit = v } }
    var sources []string
    if s := q.Get("sources"); s != "" {
        raw := strings.Split(s, ",")
        for _, r := range raw {
            t := strings.TrimSpace(r)
            if t == "" { continue }
            if t == "task" || t == "event" || t == "reminder" { sources = append(sources, t) }
        }
    }
    return ParsedCalendarParams{Year: year, Month: month, Limit: limit, Sources: sources}
}
