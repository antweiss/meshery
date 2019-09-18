package helpers

import (
	"bytes"
	"context"
	"encoding/json"
	"html/template"
	"net/url"
	"time"

	"github.com/grafana-tools/sdk"
	"github.com/layer5io/meshery/models"
	"github.com/pkg/errors"
	promAPI "github.com/prometheus/client_golang/api"
	promQAPI "github.com/prometheus/client_golang/api/prometheus/v1"
	promModel "github.com/prometheus/common/model"
	"github.com/sirupsen/logrus"
)

// PrometheusClient represents a prometheus client in Meshery
type PrometheusClient struct {
	grafanaClient *GrafanaClient
	promURL       string
}

// NewPrometheusClient returns a PrometheusClient
func NewPrometheusClient(ctx context.Context, promURL string, validate bool) (*PrometheusClient, error) {
	// client, err := promAPI.NewClient(promAPI.Config{Address: promURL})
	// if err != nil {
	// 	msg := errors.New("unable to connect to prometheus")
	// 	logrus.Error(errors.Wrap(err, msg.Error()))
	// 	return nil, msg
	// }
	// queryAPI := promQAPI.NewAPI(client)
	// return &PrometheusClient{
	// 	client:      client,
	// 	queryClient: queryAPI,
	// }, nil
	p := &PrometheusClient{
		grafanaClient: NewGrafanaClientForPrometheus(promURL),
		promURL:       promURL,
	}
	if validate {
		_, err := p.grafanaClient.makeRequest(ctx, promURL+"/api/v1/status/config")
		if err != nil {
			return nil, err
		}
	}
	return p, nil
}

// ImportGrafanaBoard takes raw Grafana board json and returns GrafanaBoard pointer for use in Meshery
func (p *PrometheusClient) ImportGrafanaBoard(ctx context.Context, boardData []byte) (*models.GrafanaBoard, error) {
	board := &sdk.Board{}
	if err := json.Unmarshal(boardData, board); err != nil {
		msg := errors.New("unable to parse grafana board data")
		logrus.Error(errors.Wrap(err, msg.Error()))
		return nil, msg
	}
	return p.grafanaClient.ProcessBoard(board, &sdk.FoundBoard{
		Title: board.Title,
		URI:   board.Slug,
	})
}

// Query queries prometheus using the GrafanaClient
func (p *PrometheusClient) Query(ctx context.Context, queryData *url.Values) ([]byte, error) {
	return p.grafanaClient.GrafanaQuery(ctx, queryData)
}

// QueryRange queries prometheus using the GrafanaClient
func (p *PrometheusClient) QueryRange(ctx context.Context, queryData *url.Values) ([]byte, error) {
	return p.grafanaClient.GrafanaQueryRange(ctx, queryData)
}

// GetStaticBoard retrieves the static board config
func (p *PrometheusClient) GetStaticBoard(ctx context.Context) (*models.GrafanaBoard, error) {
	var buf bytes.Buffer
	ttt := template.New("staticBoard").Delims("[[", "]]")
	instances, err := p.getAllNodes(ctx)
	if err != nil {
		err = errors.Wrapf(err, "unable to get all the nodes")
		logrus.Error(err)
		return nil, err
	}
	logrus.Debugf("Instances: %v, length: %d", instances, len(instances))
	tpl := template.Must(ttt.Parse(staticBoard))
	if err := tpl.Execute(&buf, map[string]interface{}{
		"instances":  instances,
		"indexCheck": len(instances) - 1,
	}); err != nil {
		err = errors.Wrapf(err, "unable to get the static board")
		logrus.Error(err)
		return nil, err
	}
	// logrus.Debugf("Board json: %s", buf.String())
	return p.ImportGrafanaBoard(ctx, buf.Bytes())
}

func (p *PrometheusClient) getAllNodes(ctx context.Context) ([]string, error) {
	// api/datasources/proxy/1/api/v1/series?match[]=node_boot_time_seconds%7Bcluster%3D%22%22%2C%20job%3D%22node-exporter%22%7D&start=1568392571&end=1568396171
	c, _ := promAPI.NewClient(promAPI.Config{
		Address: p.promURL,
	})
	qc := promQAPI.NewAPI(c)
	labelSet, _, err := qc.Series(ctx, []string{`node_boot_time_seconds{cluster="", job="node-exporter"}`}, time.Now().Add(-5*time.Minute), time.Now())
	if err != nil {
		err = errors.Wrapf(err, "unable to get the label set series")
		logrus.Error(err)
		return nil, err
	}
	result := []string{}
	for _, l := range labelSet {
		inst, _ := l["instance"]
		ins := string(inst)
		if ins != "" {
			result = append(result, ins)
		}
	}
	return result, nil
}

// QueryRangeUsingClient performs a range query within a window
func (p *PrometheusClient) QueryRangeUsingClient(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration) (promModel.Value, error) {
	c, _ := promAPI.NewClient(promAPI.Config{
		Address: p.promURL,
	})
	qc := promQAPI.NewAPI(c)
	result, _, err := qc.QueryRange(ctx, query, promQAPI.Range{
		Start: startTime,
		End:   endTime,
		Step:  step,
	})
	if err != nil {
		err := errors.Wrapf(err, "error fetching data for query: %s, with start: %v, end: %v, step: %v", query, startTime, endTime, step)
		logrus.Error(err)
		return nil, err
	}
	return result, nil
}

// ComputeStep computes the step size for a window
func (p *PrometheusClient) ComputeStep(ctx context.Context, start, end time.Time) time.Duration {
	step := 5 * time.Second
	diff := end.Sub(start)
	// all calc. here are approx.
	if diff <= 10*time.Minute { // 10 mins
		step = 5 * time.Second
	} else if diff <= 30*time.Minute { // 30 mins
		step = 10 * time.Second
	} else if diff > 30*time.Minute && diff <= time.Hour { // 60 mins/1hr
		step = 20 * time.Second
	} else if diff > 1*time.Hour && diff <= 3*time.Hour { // 3 time.Hour
		step = 1 * time.Minute
	} else if diff > 3*time.Hour && diff <= 6*time.Hour { // 6 time.Hour
		step = 2 * time.Minute
	} else if diff > 6*time.Hour && diff <= 1*24*time.Hour { // 24 time.Hour/1 day
		step = 8 * time.Minute
	} else if diff > 1*24*time.Hour && diff <= 2*24*time.Hour { // 2 24*time.Hour
		step = 16 * time.Minute
	} else if diff > 2*24*time.Hour && diff <= 4*24*time.Hour { // 4 24*time.Hour
		step = 32 * time.Minute
	} else if diff > 4*24*time.Hour && diff <= 7*24*time.Hour { // 7 24*time.Hour
		step = 56 * time.Minute
	} else if diff > 7*24*time.Hour && diff <= 15*24*time.Hour { // 15 24*time.Hour
		step = 2 * time.Hour
	} else if diff > 15*24*time.Hour && diff <= 1*30*24*time.Hour { // 30 24*time.Hour/1 month
		step = 4 * time.Hour
	} else if diff > 1*30*24*time.Hour && diff <= 3*30*24*time.Hour { // 3 months
		step = 12 * time.Hour
	} else if diff > 3*30*24*time.Hour && diff <= 6*30*24*time.Hour { // 6 months
		step = 1 * 24 * time.Hour
	} else if diff > 6*30*24*time.Hour && diff <= 1*12*30*24*time.Hour { // 1 year/12 months
		step = 2 * 24 * time.Hour
	} else if diff > 1*12*30*24*time.Hour && diff <= 2*12*30*24*time.Hour { // 2 years
		step = 4 * 24 * time.Hour
	} else if diff > 2*12*30*24*time.Hour && diff <= 5*12*30*24*time.Hour { // 5 years
		step = 10 * 24 * time.Hour
	} else {
		step = 30 * 24 * time.Hour
	}
	return step
}

// This is the output of this board: Kubernetes / Compute Resources / Cluster, which comes as part of
// prometheus operator install &
// $datasource template variable replaced with "prometheus" &
// $cluster template variable replaced with ""
const staticBoard = `
[[ $indexCheck := .indexCheck ]]
{
"annotations": {
  "list": [
	{
	  "builtIn": 1,
	  "datasource": "-- Grafana --",
	  "enable": true,
	  "hide": true,
	  "iconColor": "rgba(0, 211, 255, 1)",
	  "name": "Annotations & Alerts",
	  "type": "dashboard"
	}
  ]
},
"editable": false,
"gnetId": null,
"graphTooltip": 0,
"id": 7,
"iteration": 1568396170452,
"links": [],
"panels": [
 [[ range $ind, $instance := .instances ]]
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 12,
	  "x": 0,
	  "y": 0
	},
	"id": 2,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max(node_load1{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "load 1m",
		"refId": "A"
	  },
	  {
		"expr": "max(node_load5{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "load 5m",
		"refId": "B"
	  },
	  {
		"expr": "max(node_load15{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "load 15m",
		"refId": "C"
	  },
	  {
		"expr": "count(node_cpu_seconds_total{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\", mode=\"user\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "logical cores",
		"refId": "D"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "System load - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "short",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "short",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 12,
	  "x": 12,
	  "y": 0
	},
	"id": 3,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "sum by (cpu) (irate(node_cpu_seconds_total{cluster=\"\", job=\"node-exporter\", mode!=\"idle\", instance=\"[[ $instance ]]\"}[5m]))",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "{{cpu}}",
		"refId": "A"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Usage Per Core - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "percentunit",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "percentunit",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 18,
	  "x": 0,
	  "y": 7
	},
	"id": 4,
	"legend": {
	  "alignAsTable": true,
	  "avg": true,
	  "current": true,
	  "max": false,
	  "min": false,
	  "rightSide": true,
	  "show": true,
	  "total": false,
	  "values": true
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max (sum by (cpu) (irate(node_cpu_seconds_total{cluster=\"\", job=\"node-exporter\", mode!=\"idle\", instance=\"[[ $instance ]]\"}[2m])) ) * 100",
		"format": "time_series",
		"intervalFactor": 10,
		"legendFormat": "{{ cpu }}",
		"refId": "A"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "CPU Utilization - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "percent",
		"label": null,
		"logBase": 1,
		"max": 100,
		"min": 0,
		"show": true
	  },
	  {
		"format": "percent",
		"label": null,
		"logBase": 1,
		"max": 100,
		"min": 0,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"cacheTimeout": null,
	"colorBackground": false,
	"colorValue": false,
	"colors": [
	  "rgba(50, 172, 45, 0.97)",
	  "rgba(237, 129, 40, 0.89)",
	  "rgba(245, 54, 54, 0.9)"
	],
	"datasource": "prometheus",
	"format": "percent",
	"gauge": {
	  "maxValue": 100,
	  "minValue": 0,
	  "show": true,
	  "thresholdLabels": false,
	  "thresholdMarkers": true
	},
	"gridPos": {
	  "h": 7,
	  "w": 6,
	  "x": 18,
	  "y": 7
	},
	"id": 5,
	"interval": null,
	"links": [],
	"mappingType": 1,
	"mappingTypes": [
	  {
		"name": "value to text",
		"value": 1
	  },
	  {
		"name": "range to text",
		"value": 2
	  }
	],
	"maxDataPoints": 100,
	"nullPointMode": "connected",
	"nullText": null,
	"postfix": "",
	"postfixFontSize": "50%",
	"prefix": "",
	"prefixFontSize": "50%",
	"rangeMaps": [
	  {
		"from": "null",
		"text": "N/A",
		"to": "null"
	  }
	],
	"sparkline": {
	  "fillColor": "rgba(31, 118, 189, 0.18)",
	  "full": false,
	  "lineColor": "rgb(31, 120, 193)",
	  "show": false
	},
	"tableColumn": "",
	"targets": [
	  {
		"expr": "avg(sum by (cpu) (irate(node_cpu_seconds_total{cluster=\"\", job=\"node-exporter\", mode!=\"idle\", instance=\"[[ $instance ]]\"}[2m]))) * 100",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "",
		"refId": "A"
	  }
	],
	"thresholds": "80, 90",
	"title": "CPU Usage - [[ $instance ]]",
	"tooltip": {
	  "shared": false
	},
	"type": "singlestat",
	"valueFontSize": "80%",
	"valueMaps": [
	  {
		"op": "=",
		"text": "N/A",
		"value": "null"
	  }
	],
	"valueName": "current"
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 18,
	  "x": 0,
	  "y": 14
	},
	"id": 6,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max(node_memory_MemTotal_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}  - node_memory_MemFree_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}  - node_memory_Buffers_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}  - node_memory_Cached_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "memory used",
		"refId": "A"
	  },
	  {
		"expr": "max(node_memory_Buffers_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "memory buffers",
		"refId": "B"
	  },
	  {
		"expr": "max(node_memory_Cached_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "memory cached",
		"refId": "C"
	  },
	  {
		"expr": "max(node_memory_MemFree_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "memory free",
		"refId": "D"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Memory Usage - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"cacheTimeout": null,
	"colorBackground": false,
	"colorValue": false,
	"colors": [
	  "rgba(50, 172, 45, 0.97)",
	  "rgba(237, 129, 40, 0.89)",
	  "rgba(245, 54, 54, 0.9)"
	],
	"datasource": "prometheus",
	"format": "percent",
	"gauge": {
	  "maxValue": 100,
	  "minValue": 0,
	  "show": true,
	  "thresholdLabels": false,
	  "thresholdMarkers": true
	},
	"gridPos": {
	  "h": 7,
	  "w": 6,
	  "x": 18,
	  "y": 14
	},
	"id": 7,
	"interval": null,
	"links": [],
	"mappingType": 1,
	"mappingTypes": [
	  {
		"name": "value to text",
		"value": 1
	  },
	  {
		"name": "range to text",
		"value": 2
	  }
	],
	"maxDataPoints": 100,
	"nullPointMode": "connected",
	"nullText": null,
	"postfix": "",
	"postfixFontSize": "50%",
	"prefix": "",
	"prefixFontSize": "50%",
	"rangeMaps": [
	  {
		"from": "null",
		"text": "N/A",
		"to": "null"
	  }
	],
	"sparkline": {
	  "fillColor": "rgba(31, 118, 189, 0.18)",
	  "full": false,
	  "lineColor": "rgb(31, 120, 193)",
	  "show": false
	},
	"tableColumn": "",
	"targets": [
	  {
		"expr": "max(  (    (      node_memory_MemTotal_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}    - node_memory_MemFree_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}    - node_memory_Buffers_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}    - node_memory_Cached_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}    )    / node_memory_MemTotal_bytes{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}  ) * 100)",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "",
		"refId": "A"
	  }
	],
	"thresholds": "80, 90",
	"title": "Memory Usage - [[ $instance ]]",
	"tooltip": {
	  "shared": false
	},
	"type": "singlestat",
	"valueFontSize": "80%",
	"valueMaps": [
	  {
		"op": "=",
		"text": "N/A",
		"value": "null"
	  }
	],
	"valueName": "current"
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 12,
	  "x": 0,
	  "y": 21
	},
	"id": 8,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [
	  {
		"alias": "read",
		"yaxis": 1
	  },
	  {
		"alias": "io time",
		"yaxis": 2
	  }
	],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max(rate(node_disk_read_bytes_total{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}[2m]))",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "read",
		"refId": "A"
	  },
	  {
		"expr": "max(rate(node_disk_written_bytes_total{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}[2m]))",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "written",
		"refId": "B"
	  },
	  {
		"expr": "max(rate(node_disk_io_time_seconds_total{cluster=\"\", job=\"node-exporter\",  instance=\"[[ $instance ]]\"}[2m]))",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "io time",
		"refId": "C"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Disk I/O - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "ms",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 12,
	  "x": 12,
	  "y": 21
	},
	"id": 9,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "node:node_filesystem_usage:{cluster=\"\"}",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "{{device}}",
		"refId": "A"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Disk Space Usage - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "percentunit",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "percentunit",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 12,
	  "x": 0,
	  "y": 28
	},
	"id": 10,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max(rate(node_network_receive_bytes_total{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\", device!~\"lo\"}[5m]))",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "{{device}}",
		"refId": "A"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Network Received - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 12,
	  "x": 12,
	  "y": 28
	},
	"id": 11,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max(rate(node_network_transmit_bytes_total{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\", device!~\"lo\"}[5m]))",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "{{device}}",
		"refId": "A"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Network Transmitted - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "bytes",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"aliasColors": {},
	"bars": false,
	"dashLength": 10,
	"dashes": false,
	"datasource": "prometheus",
	"fill": 1,
	"gridPos": {
	  "h": 7,
	  "w": 18,
	  "x": 0,
	  "y": 35
	},
	"id": 12,
	"legend": {
	  "alignAsTable": false,
	  "avg": false,
	  "current": false,
	  "max": false,
	  "min": false,
	  "rightSide": false,
	  "show": true,
	  "total": false,
	  "values": false
	},
	"lines": true,
	"linewidth": 1,
	"links": [],
	"nullPointMode": "null",
	"paceLength": 10,
	"percentage": false,
	"pointradius": 5,
	"points": false,
	"renderer": "flot",
	"repeat": null,
	"seriesOverrides": [],
	"spaceLength": 10,
	"stack": false,
	"steppedLine": false,
	"targets": [
	  {
		"expr": "max(  node_filesystem_files{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}  - node_filesystem_files_free{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "inodes used",
		"refId": "A"
	  },
	  {
		"expr": "max(node_filesystem_files_free{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"})",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "inodes free",
		"refId": "B"
	  }
	],
	"thresholds": [],
	"timeFrom": null,
	"timeRegions": [],
	"timeShift": null,
	"title": "Inodes Usage - [[ $instance ]]",
	"tooltip": {
	  "shared": false,
	  "sort": 0,
	  "value_type": "individual"
	},
	"type": "graph",
	"xaxis": {
	  "buckets": null,
	  "mode": "time",
	  "name": null,
	  "show": true,
	  "values": []
	},
	"yaxes": [
	  {
		"format": "short",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  },
	  {
		"format": "short",
		"label": null,
		"logBase": 1,
		"max": null,
		"min": null,
		"show": true
	  }
	],
	"yaxis": {
	  "align": false,
	  "alignLevel": null
	}
  },
  {
	"cacheTimeout": null,
	"colorBackground": false,
	"colorValue": false,
	"colors": [
	  "rgba(50, 172, 45, 0.97)",
	  "rgba(237, 129, 40, 0.89)",
	  "rgba(245, 54, 54, 0.9)"
	],
	"datasource": "prometheus",
	"format": "percent",
	"gauge": {
	  "maxValue": 100,
	  "minValue": 0,
	  "show": true,
	  "thresholdLabels": false,
	  "thresholdMarkers": true
	},
	"gridPos": {
	  "h": 7,
	  "w": 6,
	  "x": 18,
	  "y": 35
	},
	"id": 13,
	"interval": null,
	"links": [],
	"mappingType": 1,
	"mappingTypes": [
	  {
		"name": "value to text",
		"value": 1
	  },
	  {
		"name": "range to text",
		"value": 2
	  }
	],
	"maxDataPoints": 100,
	"nullPointMode": "connected",
	"nullText": null,
	"postfix": "",
	"postfixFontSize": "50%",
	"prefix": "",
	"prefixFontSize": "50%",
	"rangeMaps": [
	  {
		"from": "null",
		"text": "N/A",
		"to": "null"
	  }
	],
	"sparkline": {
	  "fillColor": "rgba(31, 118, 189, 0.18)",
	  "full": false,
	  "lineColor": "rgb(31, 120, 193)",
	  "show": false
	},
	"tableColumn": "",
	"targets": [
	  {
		"expr": "max(  (    (      node_filesystem_files{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}    - node_filesystem_files_free{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}    )    / node_filesystem_files{cluster=\"\", job=\"node-exporter\", instance=\"[[ $instance ]]\"}  ) * 100)",
		"format": "time_series",
		"intervalFactor": 2,
		"legendFormat": "",
		"refId": "A"
	  }
	],
	"thresholds": "80, 90",
	"title": "Inodes Usage - [[ $instance ]]",
	"tooltip": {
	  "shared": false
	},
	"type": "singlestat",
	"valueFontSize": "80%",
	"valueMaps": [
	  {
		"op": "=",
		"text": "N/A",
		"value": "null"
	  }
	],
	"valueName": "current"
  }[[if ne $indexCheck $ind ]],
  [[ end ]]
  [[ end ]]
],
"refresh": "",
"schemaVersion": 18,
"style": "dark",
"tags": [
  "kubernetes-mixin"
],
"templating": {
  "list": [
	{
	  "current": {
		"selected": true,
		"text": "prometheus",
		"value": "prometheus"
	  },
	  "hide": 0,
	  "label": null,
	  "name": "datasource",
	  "options": [],
	  "query": "prometheus",
	  "refresh": 1,
	  "regex": "",
	  "skipUrlSync": false,
	  "type": "datasource"
	},
	{
	  "allValue": null,
	  "current": {
		"isNone": true,
		"selected": true,
		"text": "None",
		"value": ""
	  },
	  "datasource": "prometheus",
	  "definition": "",
	  "hide": 2,
	  "includeAll": false,
	  "label": "cluster",
	  "multi": false,
	  "name": "cluster",
	  "options": [],
	  "query": "label_values(kube_pod_info, cluster)",
	  "refresh": 2,
	  "regex": "",
	  "skipUrlSync": false,
	  "sort": 0,
	  "tagValuesQuery": "",
	  "tags": [],
	  "tagsQuery": "",
	  "type": "query",
	  "useTags": false
	},
	{
	  "allValue": null,
	  "current": {
		"selected": false,
		"tags": [],
		"text": "10.199.75.57:9100",
		"value": "10.199.75.57:9100"
	  },
	  "datasource": "prometheus",
	  "definition": "",
	  "hide": 0,
	  "includeAll": false,
	  "label": null,
	  "multi": false,
	  "name": "instance",
	  "options": [],
	  "query": "label_values(node_boot_time_seconds{cluster=\"\", job=\"node-exporter\"}, instance)",
	  "refresh": 2,
	  "regex": "",
	  "skipUrlSync": false,
	  "sort": 0,
	  "tagValuesQuery": "",
	  "tags": [],
	  "tagsQuery": "",
	  "type": "query",
	  "useTags": false
	}
  ]
},
"time": {
  "from": "now-1h",
  "to": "now"
},
"timepicker": {
  "refresh_intervals": [
	"5s",
	"10s",
	"30s",
	"1m",
	"5m",
	"15m",
	"30m",
	"1h",
	"2h",
	"1d"
  ],
  "time_options": [
	"5m",
	"15m",
	"1h",
	"6h",
	"12h",
	"24h",
	"2d",
	"7d",
	"30d"
  ]
},
"timezone": "",
"title": "Kubernetes / Nodes",
"uid": "fa49a4706d07a042595b664c87fb33ea",
"version": 1
}`
