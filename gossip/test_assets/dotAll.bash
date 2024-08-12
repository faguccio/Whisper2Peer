#!/bin/bash

stem="${1%.dot}"

for layout in circo dot fdp neato sfdp twopi ; do
	echo $layout
	dot "${stem}.dot" -K${layout} -Tsvg -o "${stem}.${layout}.svg" -Gstylesheet=${stem}.css
done
