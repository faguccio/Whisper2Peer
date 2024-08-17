.PHONY: clean spellA spellT cont count

MAIN  = main
FILES = $(MAIN).tex $(wildcard figures/*.pdf) $(wildcard figures/*.png) $(wildcard figures/*.jpg) $(wildcard figures/*.tex) Makefile

AUX = tex-aux
OPT =

.PRECIOUS: $(MAIN).pdf

$(MAIN).pdf: $(FILES) figures/class.pdf
	test -d $(AUX) || mkdir $(AUX)
	[[ "$$(whoami)" == "lukas" ]] && ./cluttex_teal --output-directory=$(AUX) --change-directory -e pdflatex $(OPT) "$<" || true
	[[ "$$(whoami)" != "lukas" ]] && tectonic "$<" || true

figures/class.pdf: figures/class.svg
	cd figures && inkscape --export-filename=class.pdf class.svg

figures/class.svg: figures/class.puml
	cd figures && plantuml -tsvg class.puml

cont: $(FILES)
	$(MAKE) OPT="--watch=inotifywait"

count:
	texcount -col -inc $(MAIN).tex

clean:
	-test -d $(AUX)  && $(RM) -r $(AUX)

spellA: $(FILES)
	-aspell --home-dir=. --personal=dict.txt -l en_US -t -c "$<"
	iconv -f ISO-8859-1 -t UTF-8 ./dict.txt > ./dict.txt2
spellT: $(FILES)
	-textidote --check en --remove tikzpicture --replace replacements.txt --dict ./dict.txt2 --output html "$<" > $(MAIN)-texidote.html
