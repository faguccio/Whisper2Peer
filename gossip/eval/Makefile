.PHONY: cont reachedAll cleanData cleanGen cleanAll distclean

MAIN  = main
all: $(MAIN).pdf reached.pdf reachedAll


#############
# DATA STUFF #
#############
CFG_STEM = ../test_assets/erdos
TEST_SCRIPT = ../cmd/testing/main.go
dist_cnt.csv dist_reached.csv packets_sent.csv reached.css: $(CFG_STEM).json
	go run "$(TEST_SCRIPT)" "$<"


#############
# TEX STUFF #
#############
FILES = $(MAIN).tex Makefile dist_cnt.csv dist_reached.csv packets_sent.csv
AUX ?= tex-aux
OPT ?= --change-directory -e lualatex

$(MAIN).pdf: $(FILES)
	test -d $(AUX) || mkdir $(AUX)
	cluttealtex --output-directory=$(AUX) $(OPT) "$<"

cont: $(FILES)
	touch "$(MAIN).tex"
	$(MAKE) OPT="$(OPT) --watch=inotifywait --memoize_opt=readonly"


#######################
# REACHED GRAPH STUFF #
#######################
reached.pdf: reached_circo.pdf
	cp "$<" "$@"

reachedAll: reached_circo.pdf reached_dot.pdf reached_fdp.pdf reached_neato.pdf reached_sfdp.pdf reached_twopi.pdf

# generate the pdf
reached_circo.pdf reached_dot.pdf reached_fdp.pdf reached_neato.pdf reached_sfdp.pdf reached_twopi.pdf: reached_%.pdf: $(CFG_STEM).dot .reached_%.css
	dot "$(CFG_STEM).dot" -K$(patsubst reached_%.pdf,%,$@) -Tsvg -o "$(patsubst %.pdf,%.svg,$@)" -Gstylesheet=$(patsubst reached_%.pdf,.reached_%.css,$@)
	chromium --headless --disable-gpu --print-to-pdf-no-header --print-to-pdf="$@" "$(patsubst %.pdf,%.svg,$@)" &>/dev/null
	# @rm "$(patsubst %.pdf,%.svg,$@)"

# render graph once to be able to obtain the size
.print_circo.svg .print_dot.svg .print_fdp.svg .print_neato.svg .print_sfdp.svg .print_twopi.svg: .print_%.svg: $(CFG_STEM).dot
	@dot "$(CFG_STEM).dot" -K$(patsubst .print_%.svg,%,$@) -Tsvg -o "$@"

# generate the css stylesheet which includes the svg size
.reached_circo.css .reached_dot.css .reached_fdp.css .reached_neato.css .reached_sfdp.css .reached_twopi.css: .reached_%.css: .print_%.svg reached.css
	@cp reached.css "$@"
	@width=$$(xq -r '.svg."@width"' "$<") ;\
		  height=$$(xq -r '.svg."@height"' "$<") ;\
		  echo "@media print {@page{margin:0mm 0mm 0mm 0mm; size:$${width} $${height};}}" >> "$@"


##################
# CLEANING STUFF #
##################

cleanAll: cleanData cleanGen

cleanGen:
	-test -d $(AUX)  && $(RM) -r $(AUX)
	-$(RM) reached_circo.svg reached_dot.svg reached_fdp.svg reached_neato.svg reached_sfdp.svg reached_twopi.svg
	-$(RM) .print* .reached*

distclean: cleanAll
	-$(RM) reached.pdf reached_circo.pdf reached_dot.pdf reached_fdp.pdf reached_neato.pdf reached_sfdp.pdf reached_twopi.pdf
	-$(RM) main.pdf

cleanData:
	-$(RM) *.css *.csv
