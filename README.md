# mandelbrotset
This program is a web application written in Go that makes extensive use of the html/template package.  Issue "go build mandelbrot.go" or issue "go run mandelbrot.go" to start the server.
In a web browser enter http://127.0.0.1:8080/mandelbrot in the address bar.  The set can be zoomed into for inspection of areas of interest.  Just enter the x and y endpoint coordinates.  The plot uses a 300 x 300 cell grid, each cell is 2px.  The shade of gray (white to black) denotes the number of interations it took the recursion z(n+1) = z(n)^2 + c to become greater than 2 in complex magnitude.  The program uses for colors (shares of gray).  The constant c is the starting point in the complex plane for the cell.  The iteration is done 200 times for each cell and there are 90,000 cells in the grid.

![image](https://user-images.githubusercontent.com/117768679/208185893-32fa9977-a55e-4647-9a47-8ae7f05a5eeb.png)
![image](https://user-images.githubusercontent.com/117768679/208186398-9384e36b-67a7-484c-92e8-dc5d6fb507f1.png)

