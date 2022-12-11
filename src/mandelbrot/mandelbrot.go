// mandelbrot plots the set of complex points that satisfy z(n+1) = z(n)^2 + c
// as n goes to infinity and the magnitude is less than 4.  c = x + yi is the
// x-y coordinate of the cell.  z(0) = c.  Zoom to any point in the complex plane.

package main

import (
	"fmt"
	"log"
	"math/cmplx"
	"net/http"
	"strconv"
	"text/template"
	"time"
)

const (
	rows          = 200                                            // #rows in grid
	columns       = 200                                            // #columns in grid
	tmpl          = "../../src/mandelbrot/templates/plotdata.html" // html template relative address
	addr          = "127.0.0.1:8080"                               // http server listen address
	pattern       = "/mandelbrot"                                  // http handler pattern for plotting data
	xlabels       = 11                                             // # labels on x axis
	ylabels       = 11                                             // # labels on y axis
	maxIterations = 200                                            // maximum iterations to determine the Mandelbrot set
	colors        = 5                                              // number of colors (shades of gray) in the Mandelbrot plot
)

// plot data that is parsed into the HTML template
type PlotT struct {
	Grid   []string // plotting grid
	Status string   // status of the plot
	Xlabel []string // x-axis labels
	Ylabel []string // y-axis labels
}

// Result sent in the channel from the goroutines
type Result struct {
	row    int
	minits int   // minimum iteration for this row
	maxits int   // maximum interation for this row
	its    []int // cell iterations for this row
}

// Plot x-y coordinate bounds supplied by the user for zooming
type Endpoints struct {
	xmin float64
	xmax float64
	ymin float64
	ymax float64
}

var (
	t *template.Template
)

// init parses the html template file done only once
func init() {
	t = template.Must(template.ParseFiles(tmpl))
}

// determineSet determines which cells are in the Mandelbrot set by
// squaring the point and requiring it to remain bounded for maxIterations.
// Return the number of iterations done before escaping the bounds.
func determineSet(row int, col int, ep *Endpoints) int {

	x := float64(col)/float64(columns-1)*(ep.xmax-ep.xmin) + ep.xmin
	y := ep.ymax - float64(row)/float64(rows-1)*(ep.ymax-ep.ymin)
	z := complex(x, y) // initial value

	var v complex128
	for n := 0; n < maxIterations; n++ {
		v = v*v + z
		if cmplx.Abs(v) > 2 {
			return n
		}
	}
	return maxIterations
}

// processRow determines which cells in the row are in the Mandelbrot set
func processRow(row int, result chan<- Result, ep *Endpoints) {
	// Loop over the columns (cells) and find those that satisfy Mandelbrot
	// The number of iterations to escape is returned.
	res := Result{}
	res.its = make([]int, columns)
	res.row = row

	for col := 0; col < columns; col++ {
		its := determineSet(row, col, ep)
		if its > res.maxits {
			res.maxits = its
		}
		if its < res.minits {
			res.minits = its
		}
		res.its[col] = its
	}

	// Send the result back
	result <- res
}

// handlePlotting receives the complex plane endpoints to inspect and plots the
// the Mandelbrot iteration results.
func handlePlotting(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Printf("Start Time: %v\n", start.Format(time.RFC850))
	var (
		plot      PlotT
		xmax      float64 = .8 // default endpoints in complex plane
		xmin      float64 = -1.6
		ymax      float64 = 1.2
		ymin      float64 = -1.2
		endpoints Endpoints
	)

	plot.Grid = make([]string, rows*columns)
	plot.Xlabel = make([]string, xlabels)
	plot.Ylabel = make([]string, ylabels)

	// channel for receiving results from goroutines
	result := make(chan Result)

	xstart := r.FormValue("xstart")
	xend := r.FormValue("xend")
	ystart := r.FormValue("ystart")
	yend := r.FormValue("yend")
	if len(xstart) > 0 && len(xend) > 0 &&
		len(ystart) > 0 && len(yend) > 0 {
		x1, err1 := strconv.ParseFloat(xstart, 64)
		x2, err2 := strconv.ParseFloat(xend, 64)
		y1, err3 := strconv.ParseFloat(ystart, 64)
		y2, err4 := strconv.ParseFloat(yend, 64)

		if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
			plot.Status = "x or y values are not numbers."
			fmt.Printf("error: x start error = %v, x end error = %v\n", err1, err2)
			fmt.Printf("error: y start error = %v, y end error = %v\n", err3, err4)
		} else {
			if (x1 < xmin || x1 > xmax) || (x2 < xmin || x2 > xmax) || (x1 >= x2) {
				plot.Status = "values are not in x range."
				fmt.Printf("error: start or end value not in x range.\n")
			} else if (y1 < ymin || y1 > ymax) || (y2 < ymin || y2 > ymax) || (y1 >= y2) {
				plot.Status = "values are not in y range."
				fmt.Printf("error: start or end value not in y range.\n")
			} else {
				// Valid endpoints, replace the default min and max values
				xmin = x1
				xmax = x2
				ymin = y1
				ymax = y2
			}
		}
	}

	endpoints = Endpoints{xmin, xmax, ymin, ymax}

	for row := 0; row < rows; row++ {
		// process each row in a goroutine
		go processRow(row, result, &endpoints)
	}

	// Collect the results from the goroutines
	maxits := 0
	minits := maxIterations
	for row := 0; row < rows; row++ {
		result := <-result
		if result.minits < minits {
			minits = result.minits
		}
		if result.maxits > maxits {
			maxits = result.maxits
		}

		// Save the interations of all the cells in this row
		for col := 0; col < columns; col++ {
			plot.Grid[result.row*columns+col] = strconv.Itoa(result.its[col])
		}
	}

	// Map interations to background color:  higher iterations are dark gray to black,
	// lower interations are white to lighter shades of gray.  Black denotes members
	// of the set.
	color := []string{"gray1", "gray2", "gray3", "gray4", "gray5"}

	// scale for iterations to color
	its2color := float64(len(color)-1) / float64(maxits-minits)

	// Set the background color for all the cells in the grid based on cell iteration
	for i, its := range plot.Grid {
		itn, err := strconv.Atoi(its)
		if err != nil {
			fmt.Printf("strconv iterations error for index %d:  %v\n", i, err)
			// color this cell as not being in the set
			itn = minits
		}
		plot.Grid[i] = color[int(float64(itn-minits)*its2color+.5)]
	}

	// Construct x-axis labels
	incr := (xmax - xmin) / (xlabels - 1)
	x := xmin
	// First label is empty for alignment purposes
	for i := range plot.Xlabel {
		plot.Xlabel[i] = fmt.Sprintf("%.2f", x)
		x += incr
	}

	// Construct the y-axis labels
	incr = (ymax - ymin) / (ylabels - 1)
	y := ymin
	for i := range plot.Ylabel {
		plot.Ylabel[i] = fmt.Sprintf("%.2f", y)
		y += incr
	}

	plot.Status = fmt.Sprintf("Status: Data plotted from (%v,%v) to (%v,%v)", xmin, ymin, xmax, ymax)

	// Write to HTTP using template and grid
	if err := t.Execute(w, plot); err != nil {
		log.Fatalf("Write to HTTP output using template with grid error: %v\n", err)
	}
	end := time.Now()
	fmt.Printf("End Time: %v\n", end.Format(time.RFC850))
	fmt.Printf("Elapsed time: %v\n", time.Since(start))
}

// executive program
func main() {
	// Setup http server with handler for reading form and plotting points
	http.HandleFunc(pattern, handlePlotting)
	// Setup http server with handler for generating data for testing
	http.ListenAndServe(addr, nil)
}
