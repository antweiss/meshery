import { Component } from 'react';
import PropTypes from 'prop-types';
import { withStyles } from '@material-ui/core/styles';
import { NoSsr, IconButton, Card, CardContent, Typography, CardHeader } from '@material-ui/core';
import { updateProgress } from '../lib/store';
import {connect} from "react-redux";
import { bindActionCreators } from 'redux';
import dataFetch from '../lib/data-fetch';
import { withSnackbar } from 'notistack';
import { Chart, Line } from 'react-chartjs-2';
import moment from 'moment';
import OpenInNewIcon from '@material-ui/icons/OpenInNewOutlined';
import GrafanaCustomGaugeChart from './GrafanaCustomGaugeChart';

let c3;
if (typeof window !== 'undefined') { 
  c3 = require('c3');
}
const grafanaStyles = theme => ({
    root: {
      width: '100%',
    },
    column: {
      flexBasis: '33.33%',
    },
    heading: {
      fontSize: theme.typography.pxToRem(15),
    },
    secondaryHeading: {
      fontSize: theme.typography.pxToRem(15),
      color: theme.palette.text.secondary,
    },
    dateRangePicker: {
      display: 'flex',
      justifyContent: 'flex-end',
      marginRight: theme.spacing(1),
      marginBottom: theme.spacing(2),
    },
    // iframe: {
    //   minHeight: theme.spacing(55),
    //   minWidth: theme.spacing(55),
    // },
    chartjsTooltip: {
      position: 'absolute',
      background: 'rgba(0, 0, 0, .7)',
      color: 'white',
      borderRadius: '3px',
      transition: 'all .1s ease',
      pointerEvents: 'none',
      transform: 'translate(-50%, 0)',
    },
    chartjsTooltipKey: {
			display: 'inline-block',
			width: '10px',
			height: '10px',
			marginRight: '10px',
    },
    cardHeader: {
      fontSize: theme.spacing(2),
    },
    error: {
      color: '#D32F2F',
      width: '100%',
      textAlign: 'center',
      fontSize: '12px',
      // fontFamily: 'Helvetica Nueue',
      fontWeight: 'bold',
    },
  });

const grafanaDateRangeToDate = (dt, startDate) => {
  let dto = new Date();
  switch (dt) {
    case 'now-2d':
        dto.setDate(dto.getDate() - 2);
        break;
    case 'now-7d':
        dto.setDate(dto.getDate() - 7);
        break;
    case 'now-30d':
        dto.setDate(dto.getDate() - 30);
        break;
    case 'now-90d':
        dto.setDate(dto.getDate() - 90);
        break;
    case 'now-6M':
        dto.setMonth(dto.getMonth() - 6);
        break;
    case 'now-1y':
        dto.setFullYear(dto.getFullYear() - 1);
        break;
    case 'now-2y':
        dto.setFullYear(dto.getFullYear() - 2);
        break;
    case 'now-5y':
        dto.setFullYear(dto.getFullYear() - 5);
        break;
    case 'now-1d/d':
        dto.setDate(dto.getDate() - 1);
        if(startDate){
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now-2d/d':
        dto.setDate(dto.getDate() - 2);
        if(startDate){
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now-7d/d':
        dto.setDate(dto.getDate() - 7);
        if(startDate){
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now-1w/w':
        dto.setDate(dto.getDate() - 6 - (dto.getDay() + 8) % 7);
        if(startDate){
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setDate(dto.getDate() + 6);
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now-1M/M':
        dto.setMonth(dto.getMonth() - 1);
        if(startDate){
          dto.setDate(1);
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setMonth(dto.getMonth());
          dto.setDate(0);
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now-1y/y':
        dto.setFullYear(dto.getFullYear() - 1);
        if(startDate){
          dto.setMonth(0);
          dto.setDate(1);
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setMonth(12);
          dto.setDate(0);
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now/d':
        dto.setDate(dto.getDate() - 6 - (dto.getDay() + 8) % 7);
        if(startDate){
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now':
        break;
    case 'now/w':
        dto.setDate(dto.getDate() - (dto.getDay() + 7) % 7);
        if(startDate){
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setDate(dto.getDate() + 6);
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now/M':
        if(startDate){
          dto.setDate(1);
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setMonth(dto.getMonth()+1);
          dto.setDate(0);
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now/y':
        if(startDate){
          dto.setMonth(0);
          dto.setDate(1);
          dto.setHours(0);
          dto.setMinutes(0);
          dto.setSeconds(0);
          dto.setMilliseconds(0);
        } else {
          dto.setMonth(12);
          dto.setDate(0);
          dto.setHours(23);
          dto.setMinutes(59);
          dto.setSeconds(59);
          dto.setMilliseconds(999);
        }
        break;
    case 'now-5m':
        dto.setMinutes(dto.getMinutes() - 5);
        break;
    case 'now-15m':
        dto.setMinutes(dto.getMinutes() - 15);
        break;
    case 'now-30m':
        dto.setMinutes(dto.getMinutes() - 30);
        break;
    case 'now-1h':
        dto.setHours(dto.getHours() - 1);
        break;
    case 'now-3h':
        dto.setHours(dto.getHours() - 3);
        break;
    case 'now-6h':
        dto.setHours(dto.getHours() - 6);
        break;
    case 'now-12h':
        dto.setHours(dto.getHours() - 12);
        break;
    case 'now-24h':
        dto.setHours(dto.getHours() - 24);
        break;
    default:
      return new Date(parseFloat(dt));
  }
  return dto;
}

class GrafanaCustomChart extends Component {
    constructor(props){
      super(props);
      this.chartRef = null;
      this.chart = null;
      this.timeFormat = 'MM/DD/YYYY HH:mm:ss';
      this.c3TimeFormat = '%Y-%m-%d %H:%M:%S';
      this.panelType = '';
      switch(props.panel.type){
        case 'graph':
          this.panelType = props.panel.type;
          break;
        case 'singlestat':
          this.panelType = props.panel.type ==='singlestat' && props.panel.sparkline && props.panel.sparkline.show === true?'sparkline':'gauge';
          break;
      }
      
      this.datasetIndex = {};
      this.state = {
        xAxis: [],
        chartData: [],
        error: '',
      };
    }

    componentDidMount() {
      this.configChartData();
    }

    configChartData = () => {
      const { panel, refresh, liveTail } = this.props;
      const self = this;
      // self.createOptions();
      if(panel.targets){
        panel.targets.forEach((target, ind) => {
          self.datasetIndex[`${ind}_0`] = ind;
        }); 
      }
      if(typeof self.interval !== 'undefined'){
        clearInterval(self.interval);
      }
      if(liveTail){
        self.interval = setInterval(function(){
          // self.createOptions();
          self.collectChartData();
        }, self.computeRefreshInterval(refresh)*1000);
      }
      // self.createOptions();
      self.collectChartData();
    }

    getOrCreateIndex(datasetInd) {
      if(typeof this.datasetIndex[datasetInd] !== 'undefined'){
        return this.datasetIndex[datasetInd];
      }
      let max = 0;
      Object.keys(this.datasetIndex).forEach(i => {
        if(this.datasetIndex[i] > max){
          max = this.datasetIndex[i];
        }
      });
      this.datasetIndex[datasetInd] = max+1;
      return max+1;
    }

    collectChartData = (chartInst) => {
      const { panel } = this.props;
      const self = this;
      if(panel.targets){
        panel.targets.forEach((target, ind) => {
          self.getData(ind, target, chartInst);
        });
      }
    }

    computeStep = (start, end) => {
      let step = 10;
      const diff = end-start;
      const min = 60;
      const hrs = 60*min;
      const days = 24*hrs;
      const month = 30*days; // approx.
      const year = 12*month; // approx.

      if (diff <= 30*min){ // 30 mins
        step = 10;
      } else if (diff > 30*min && diff <= 1*hrs ){ // 60 mins/1hr
        step = 20;
      } else if (diff > 1*hrs && diff <= 3*hrs) { // 3 hrs
        step = 1*min;
      } else if (diff > 3*hrs && diff <= 6*hrs) { // 6 hrs
        step = 2*min;
      } else if (diff > 6*hrs && diff <= 1*days) { // 24 hrs/1 day
        step = 8*min;
      } else if (diff > 1*days && diff <= 2*days) { // 2 days
        step = 16*min;
      } else if (diff > 2*days && diff <= 4*days) { // 4 days
        step = 32*min;
      } else if (diff > 4*days && diff <= 7*days) { // 7 days
        step = 56*min;
      } else if (diff > 7*days && diff <= 15*days) { // 15 days
        step = 2*hrs;
      } else if (diff > 15*days && diff <= 1*month) { // 30 days/1 month
        step = 4*hrs;
      } else if (diff > 1*month && diff <= 3*month) { // 3 months
        step = 12*hrs;
      } else if (diff > 3*month && diff <= 6*month) { // 6 months
        step = 1*days;
      } else if (diff > 6*month && diff <= 1*year) { // 1 year/12 months
        step = 2*days;
      } else if (diff > 1*year && diff <= 2*year) { // 2 years
        step = 4*days;
      } else if (diff > 2*year && diff <= 5*year) { // 5 years
        step = 10*days;
      } else {
        step = 30*days;
      }
      return step;
    }

    getData = async (ind, target, chartInst) => {
      const {prometheusURL, grafanaURL, panel, from, to, templateVars} = this.props;
      const {chartData} = this.state;
      let {xAxis} = this.state;

      // const cd = (typeof chartInst === 'undefined'?chartData:chartInst.data);
      let queryRangeURL = '';
      if (prometheusURL && prometheusURL !== ''){
        queryRangeURL = `/api/prometheus/query_range`;
      } else if (grafanaURL && grafanaURL !== ''){
        // grafanaURL = grafanaURL.substring(0, grafanaURL.length - 1);
        queryRangeURL = `/api/grafana/query_range`;
      }
      const self = this;
      let expr = target.expr;
      templateVars.forEach(tv => {
        const tvrs = tv.split('=');
        if (tvrs.length == 2){
          expr = expr.replace(new RegExp(`$${tvrs[0]}`.replace(/[-\/\\^$*+?.()|[\]{}]/g, '\\$&'), 'g'),tvrs[1]);
        }
      });
      
      const start = Math.round(grafanaDateRangeToDate(from).getTime()/1000);
      const end = Math.round(grafanaDateRangeToDate(to).getTime()/1000);
      const queryParams = `ds=${panel.datasource}&query=${encodeURIComponent(expr)}&start=${start}&end=${end}&step=${self.computeStep(start, end)}`;
      // TODO: need to check if it is ok to use datasource name instead of ID
                
      dataFetch(`${queryRangeURL}?${queryParams}`, { 
        method: 'GET',
        credentials: 'include',
        // headers: headers,
      }, result => {
        self.props.updateProgress({showProgress: false});
        if (typeof result !== 'undefined'){
          const fullData = self.transformDataForChart(result, target);
          xAxis = ['x'];
          fullData.forEach(({metric, data}, di) => {
            const datasetInd = self.getOrCreateIndex(`${ind}_${di}`);
            const newData = [];

            // if (typeof cd.labels[datasetInd] === 'undefined' || typeof cd.datasets[datasetInd] === 'undefined'){
              let legend = typeof target.legendFormat !== 'undefined'?target.legendFormat:'';
              if(legend === '') {
                legend = Object.keys(metric).length > 0?JSON.stringify(metric):'';
              } else{
                Object.keys(metric).forEach(metricKey => {
                  legend = legend.replace(`{{${metricKey}}}`, metric[metricKey])
                              .replace(`{{ ${metricKey} }}`, metric[metricKey]);
                });
                legend = legend.replace(`{{ `, '').replace(`{{`, '')
                            .replace(` }}`, '').replace(`}}`, '');
              }
              // cd.labels[datasetInd] = legend;
              newData.push(legend);
              // cd.datasets[datasetInd] = {
              //   label: legend,
              //   data: [],
              //   pointRadius: 0,
              //   // fill: false,
              //   fill: true,
              // };
              if(self.panelType === 'sparkline' && panel.sparkline && panel.sparkline.lineColor && panel.sparkline.fillColor){
                cd.datasets[datasetInd].borderColor = panel.sparkline.lineColor;
                cd.datasets[datasetInd].backgroundColor = panel.sparkline.fillColor;
              }
            // }
            data.forEach(({x, y}) => {
              newData.push(y);
              xAxis.push(new Date(x));
              // let toadd = true;
              // cd.datasets[datasetInd].data.forEach(({x: x1, y: y1}) => {
              //   if(x === x1) {
              //     toadd = false;
              //   }
              // });
              // if(toadd){
              //   newData.push({x, y});  
              // }
            });
            // Array.prototype.push.apply(cd.datasets[datasetInd].data, newData);
            // cd.datasets[datasetInd].data.sort((a, b) => {
            //   return new Date(a.x).getTime() - new Date(b.x).getTime();
            // })
            chartData[datasetInd] = newData;
          });
          let groups = [];
          if (typeof panel.stack !== 'undefined' && panel.stack){
            const panelGroups = [];
            chartData.forEach(y => {
              if(y.length > 0){
                panelGroups.push(y[0]); // just the label
              }
            });
            groups = [panelGroups];
          }
          // self.chart.load({
          //   x: 'x',
          //   columns: [xAxis, ...chartData],
          //   groups,
          //   type: 'area',
          // });
          self.createOptions(xAxis, chartData, groups);
          self.setState({xAxis, chartData, error:''}); // Not TOO SURE IF THIS IS NEEDED
          // if(typeof chartInst === 'undefined'){
          //   for(let cddi=0;cddi < cd.datasets.length; cddi++){
          //     if(typeof cd.datasets[cddi] === 'undefined'){
          //       cd.datasets[cddi] = {data:[], label: ''};
          //     }
          //   }
          //   self.setState({chartData, options: self.createOptions(), error:''});
          // } else {
          //   chartInst.update({
          //     preservation: true,
          //   });
          // }
        }
      }, self.handleError);
    }

    transformDataForChart(data, target) {
      if (data && data.status === 'success' && data.data && data.data.resultType && data.data.resultType === 'matrix' 
          && data.data.result && data.data.result.length > 0){
            let fullData = [];
            data.data.result.forEach(r => {
              const localData = r.values.map(arr => {
                const x = arr[0] * 1000;
                const y = parseFloat(parseFloat(arr[1]).toFixed(2));
                return {
                  x,
                  y,
                };
              })
              fullData.push({
                data: localData,
                metric: r.metric,
              })
            })
            return fullData;
      }
      return [];
    }

    showMessageInChart(){
      var self = this;
      return function(chart) {
        // const {error} = this.state;
        // if (chart.data.datasets.length === 0) {
        var ctx = chart.chart.ctx;
        var width = chart.chart.width;
        if(self.state.error !== ''){
          var height = 5 //chart.chart.height;
          // chart.clear();
          
          // ctx.save();
          ctx.textAlign = 'center';
          ctx.textBaseline = 'middle';
          ctx.font = "bold 16px 'Helvetica Nueue'";
          // ctx.fillText(chart.options.title.text, width / 2, 18);
          ctx.fillStyle = '#D32F2F';
          ctx.fillText(`There was an error communicating with the server`, width/2, 20);
          ctx.restore();
        } 
        if (self.panelType && self.panelType === 'sparkline'){
          if(chart.data.datasets.length > 0 && chart.data.datasets[0].data.length > 0){
            const dind = chart.data.datasets[0].data.length - 1;
            // ctx.fillStyle = '#D32F2F';
            const val = chart.data.datasets[0].data[dind].y;
            const unit = chart.options.scales.yAxes[0].scaleLabel.labelString;
            let msg = '';
            if (!isNaN(val)){
              if(unit.startsWith('percent')){
                const mulFactor = unit.toLowerCase() === 'percentunit'?100:1;
                msg = `${(val*mulFactor).toFixed(2)}%`;
              } else {
                msg = `${val} ${unit}`;
              }
            }

            var height = chart.chart.height;
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.font = "bold 36px 'Helvetica Nueue'";
            ctx.fillStyle = '#000000';
            ctx.fillText(msg, width/2, height/2);
            ctx.restore();
          }
        }
      }
    }

    updateDateRange(){
      const self = this;
      let tm;
      return  function({chart}){
        if (typeof tm !== 'undefined'){
          clearTimeout(tm);
        }
        if(typeof chart !== 'undefined'){
          let min = chart.scales["x-axis-0"].min;
          let max = chart.scales["x-axis-0"].max;
          tm = setTimeout(function(){
            if(!isNaN(min) && !isNaN(max)){
              min = Math.floor(min);
              max = Math.floor(max);
              self.props.updateDateRange(`${min}`, new Date(min), `${max}`, new Date(max), false, self.props.refresh);
            } else {
              self.props.updateDateRange(self.props.from, self.props.startDate, self.props.to, self.props.endDate, self.props.liveTail, self.props.refresh);  
            }
          }, 1000);
        }
        return false;
      }
    }

    createOptions(xAxis, chartData, groups) {
      const {panel, from, to} = this.props;
      // const {xAxis, chartData} = this.state;
      const fromDate = grafanaDateRangeToDate(from);
      const toDate = grafanaDateRangeToDate(to);
      const self = this;

      const showAxis = panel.type ==='singlestat' && panel.sparkline && panel.sparkline.show === true?false:true;

      const xAxes = {
        type: 'timeseries',
        show: showAxis,
        tick: {
            format: self.c3TimeFormat,
        }
      }; 

      const yAxes = {
        show: showAxis,
      };
      
      if(panel.yaxes){
        panel.yaxes.forEach(ya => {
          if(typeof ya.label !== 'undefined' && ya.label !== null){
            yAxes.label = ya.label;
          }
          if(ya.format.toLowerCase().startsWith('percent')){
            const mulFactor = ya.format.toLowerCase() === 'percentunit'?100:1;
            // yAxes.ticks = {
            //   callback: function(tick) {
            //     const tk = (tick * mulFactor).toFixed(2);
            //     return `${tk}%`;
            //   }
            // }
            yAxes.tick = {
              format: function (d) { 
                const tk = (d * mulFactor).toFixed(2);
                return `${tk}%`;
              }
            }
          }
        });
      }
      // if(self.panelType === 'sparkline'){
      //   yAxes.scaleLabel = {
      //     display: false,
      //     labelString: panel.format,
      //   };
      // }
      // const xAxes = {
      //   type: 'time',
      //   display: showAxis,
      //   time: {
      //     min: fromDate.getTime(),
      //     max: toDate.getTime(),
      //   },
      // };

      let grid = {
        // x: {
        //     show: showAxis,
        // },
        // y: {
        //     show: showAxis,
        // }
      };

      // panel.xaxes.forEach(ya => {
      //   if(ya.label !== null){
      //     yAxes.scaleLabel = {
      //       display: true,
      //       labelString: ya.label,
      //     };
      //   }
      // });
      let shouldDisplayLegend = Object.keys(this.datasetIndex).length <= 10?true:false;
      if(panel.type !== 'graph'){
        shouldDisplayLegend = false;
      }


      self.chart = c3.generate({
        // oninit: function(args){
        //   console.log(JSON.stringify(args));
        // },
        bindto: self.chartRef,
        data: {
          x: 'x',
          xFormat: self.c3TimeFormat,
          columns: [xAxis, ...chartData],
          groups,
          type: 'area',
        },
        axis: {
          x: xAxes,
          y: yAxes,
        },
        grid: grid,
        legend: {
          show: shouldDisplayLegend,
        },
        point: {
          show: false
        }
      });
    }

    componentWillUnmount(){
      if(typeof this.interval !== 'undefined'){
        clearInterval(this.interval);
      }
    }

    computeRefreshInterval = (refresh) => {
      refresh = refresh.toLowerCase();
      const l = refresh.length;
      const dur = refresh.substring(l - 1, l);
      refresh = refresh.substring(0, l - 1);
      let val = parseInt(refresh);
      switch (dur){
        case 'd':
          val *= 24;
        case 'h':
          val *= 60;
        case 'm':
          val *= 60;
        case 's':
          return val;
      }
      return 30; // fallback
    }
  
    handleError = error => {
      const self = this;
      this.props.updateProgress({showProgress: false});
      this.setState({error: error.message && error.message !== ''?error.message:(error !== ''?error:'')});
      // this.props.enqueueSnackbar(`There was an error communicating with Grafana`, {
      //   variant: 'error',
      //   action: (key) => (
      //     <IconButton
      //           key="close"
      //           aria-label="Close"
      //           color="inherit"
      //           onClick={() => self.props.closeSnackbar(key) }
      //         >
      //           <CloseIcon />
      //     </IconButton>
      //   ),
      //   autoHideDuration: 1000,
      // });
    }
    
    render() {
      const { classes, board, panel, inDialog, handleChartDialogOpen } = this.props;
      const {error} = this.state;
      let finalChartData = {
        datasets: [],
        labels: [],
      }
      // const filteredData = chartData.datasets.filter(x => typeof x !== 'undefined')
      // if(chartData.datasets.length === filteredData.length){
      //   finalChartData = chartData;
      // }
      let self = this;
      let iconComponent = (<IconButton
          key="chartDialog"
          aria-label="Open chart in a dialog"
          color="inherit"
          onClick={() => handleChartDialogOpen(board, panel) }
        >
          <OpenInNewIcon className={classes.cardHeader} />
        </IconButton>);
      
      let mainChart;
      if(this.panelType === 'gauge'){
        mainChart = (
          <GrafanaCustomGaugeChart
              data={finalChartData}
              panel={panel}
              error={error}
            />
        );
      } else {
        mainChart = (
          <div>
            <div className={classes.error}>{error && 'There was an error communicating with the server'}</div>
            <div ref={ch => self.chartRef = ch} className={classes.root}></div>
          </div>
        );
      }
      return (
        <NoSsr>
          <Card>
          {!inDialog && <CardHeader
            disableTypography={true}
            title={panel.title}
            action={iconComponent}
            className={classes.cardHeader}
          />}
          <CardContent>
            {mainChart}
          </CardContent>
          </Card>
        </NoSsr>
      );
    }
}

GrafanaCustomChart.propTypes = {
  classes: PropTypes.object.isRequired,
  grafanaURL: PropTypes.string.isRequired,
  grafanaAPIKey: PropTypes.string.isRequired,
  board: PropTypes.object.isRequired,
  panel: PropTypes.object.isRequired,
  templateVars: PropTypes.array.isRequired,
  updateDateRange: PropTypes.func.isRequired,
  handleChartDialogOpen: PropTypes.func.isRequired,
  inDialog: PropTypes.bool.isRequired,
};

const mapDispatchToProps = dispatch => {
  return {
      updateProgress: bindActionCreators(updateProgress, dispatch),
  }
}

export default withStyles(grafanaStyles)(connect(
  null,
  mapDispatchToProps
)(withSnackbar(GrafanaCustomChart)));