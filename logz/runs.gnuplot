set term png size 1200,900
set output "runs.png"

set title  'Execution time over prefetch channel depth'
set xlabel 'blocks'
set ylabel 'ms'

set origin 0.0,0.0
set logscale y

plot 'runs.csv' u 1:2  w lp title 'block read',    \
     'runs.csv' u 1:3  w lp title 'excecuteBlock', \
     'runs.csv' u 1:4  w lp title 'read account',  \
     'runs.csv' u 1:5  w lp title 'read code',     \
     'runs.csv' u 1:6  w lp title 'read storage'


