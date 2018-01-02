package imgtmpl

import (
	"bytes"
	"container/list"
	"fmt"
	"image"
	"image/color"
	"log"
	"strings"

	"github.com/alecthomas/template"
	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

func init() {
	componentFactories["text_block"] = func() Component {
		return new(TextBlock)
	}
}

type TextBlock struct {
	Bounds    image.Rectangle
	Spans     []*TextSpan
	Alignment TextBlockAlignment
	Font      *TextFont
	runes     []textRune
	lines     [][]textRune
}

type TextBlockAlignment struct {
	LineHeight int
	MaxLines   int
	Horizontal string
	Vertical   string
}

type TextFont struct {
	Name  string
	Size  int
	Color string
	font  *truetype.Font
	color color.Color
	face  font.Face
}

type TextSpan struct {
	Text string
	tmpl *template.Template
	Font *TextFont
}

type textRuneMetrics struct {
	ascent  fixed.Int26_6
	descent fixed.Int26_6
	advance fixed.Int26_6
}

type textRune struct {
	r       rune
	font    *TextFont
	metrics textRuneMetrics
	point   fixed.Point26_6
}

func (block *TextBlock) Init(_ Resources) error {
	if block.Font == nil {
		return fmt.Errorf("缺少字体配置")
	}
	if err := block.Font.init(); err != nil {
		return err
	}
	for _, span := range block.Spans {
		if err := span.init(); err != nil {
			return err
		}
	}
	if enableLog {
		log.Println("TextBlock初始化成功:", *block)
	}
	return nil
}

func (block *TextBlock) Render(dst draw.Image, params map[string]string) error {
	if err := block.splitRunes(params); err != nil {
		return err
	}
	if enableLog {
		log.Println("Rune拆分结果:", block.runes)
	}
	if err := block.splitLines(); err != nil {
		return err
	}
	if enableLog {
		log.Println("Line拆分结果:", block.lines)
	}
	if err := block.arrangeRunes(); err != nil {
		return err
	}
	if err := block.drawRunes(dst); err != nil {
		return err
	}
	return nil
}

func (block *TextBlock) splitRunes(params map[string]string) error {
	runeList := list.New()
	for _, span := range block.Spans {
		spanRuneList, err := span.splitRunes(block.Font, params)
		if err != nil {
			return err
		}
		runeList.PushBackList(spanRuneList)
	}
	block.runes = make([]textRune, runeList.Len())
	for i, r := 0, runeList.Front(); r != nil; i, r = i+1, r.Next() {
		block.runes[i] = r.Value.(textRune)
	}
	return nil
}

func (block *TextBlock) splitLines() error {
	pos := 0
	textLen := len(block.runes)
	advanceLimit := fixed.I(block.Bounds.Dx())
	lines := list.New()
	for l := 0; l < block.Alignment.MaxLines && pos < textLen; l++ {
		begin := pos
		end := begin
		advance := fixed.I(0)
		for ; end < textLen && advance+block.runes[end].metrics.advance < advanceLimit; end++ {
			advance += block.runes[end].metrics.advance
		}
		lines.PushBack(block.runes[begin:end])
		pos = end
	}
	block.lines = make([][]textRune, lines.Len())
	for i, l := 0, lines.Front(); l != nil; i, l = i+1, l.Next() {
		block.lines[i] = l.Value.([]textRune)
	}
	return nil
}

func (block *TextBlock) arrangeRunes() error {
	marginTop := block.Bounds.Min.Y
	if block.Alignment.Vertical == "middle" {
		marginTop += (block.Bounds.Dy() - block.Alignment.LineHeight*block.Alignment.MaxLines) / 2
	} else if block.Alignment.Vertical == "bottom" {
		marginTop += block.Bounds.Dy() - block.Alignment.LineHeight*block.Alignment.MaxLines
	}
	for i, line := range block.lines {
		if err := block.arrangeLineRunes(line, marginTop+i*block.Alignment.LineHeight); err != nil {
			return err
		}
	}
	return nil
}

func (block *TextBlock) arrangeLineRunes(runes []textRune, marginTop int) error {
	ascent, descent, advance := fixed.I(0), fixed.I(0), fixed.I(0)
	for _, r := range runes {
		if ascent < r.metrics.ascent {
			ascent = r.metrics.ascent
		}
		if descent < r.metrics.descent {
			descent = r.metrics.descent
		}
		advance += r.metrics.advance
	}
	height := (ascent + descent).Ceil()
	marginTop += (block.Alignment.LineHeight - height) / 2
	width := advance.Ceil()
	marginLeft := (block.Bounds.Dx() - width) / 2
	if block.Alignment.Horizontal == "left" {
		marginLeft = 0
	} else if block.Alignment.Horizontal == "right" {
		marginLeft = block.Bounds.Dx() - width
	}
	x := block.Bounds.Min.X + marginLeft
	y := marginTop + ascent.Ceil()
	for i, r := range runes {
		runes[i].point = fixed.P(x, y)
		x += r.metrics.advance.Ceil()
	}
	return nil
}

func (block *TextBlock) drawRunes(dst draw.Image) error {
	var font *TextFont
	var point fixed.Point26_6
	var buf bytes.Buffer
	for _, line := range block.lines {
		for _, r := range line {
			if font != r.font {
				if font != nil && buf.Len() > 0 {
					if err := block.drawString(dst, buf.String(), font, point); err != nil {
						return err
					}
				}
				buf.Reset()
				font = r.font
				point = r.point
			}
			buf.WriteRune(r.r)
		}
		if buf.Len() > 0 {
			if err := block.drawString(dst, buf.String(), font, point); err != nil {
				return err
			}
		}
		buf.Reset()
	}
	return nil
}

func (block *TextBlock) drawString(dst draw.Image, text string, font *TextFont, point fixed.Point26_6) error {
	context := freetype.NewContext()
	context.SetDst(dst)
	context.SetClip(block.Bounds)
	context.SetFont(font.font)
	context.SetFontSize(float64(font.Size))
	context.SetSrc(image.NewUniform(font.color))
	if _, err := context.DrawString(text, point); err != nil {
		return err
	}
	return nil
}

func (font *TextFont) init() (err error) {
	if font.color, err = parseColor(font.Color); err != nil {
		return
	}
	if font.font, err = getFont(font.Name); err != nil {
		return
	}
	font.face = truetype.NewFace(font.font, &truetype.Options{Size: float64(font.Size)})
	if enableLog {
		log.Println("初始化字体完成:", *font)
	}
	return
}

func (span *TextSpan) init() (err error) {
	if strings.Index(span.Text, "{{") >= 0 {
		span.tmpl = template.New(span.Text)
		if _, err = span.tmpl.Parse(span.Text); err != nil {
			return err
		}
	}
	if span.Font != nil {
		if err = span.Font.init(); err != nil {
			return err
		}
	}
	return nil
}

func (span *TextSpan) splitRunes(baseFont *TextFont, params map[string]string) (*list.List, error) {
	runes := list.New()
	text := span.Text
	if span.tmpl != nil {
		var buf bytes.Buffer
		if err := span.tmpl.Execute(&buf, params); err != nil {
			return nil, err
		}
		text = buf.String()
	}
	font := baseFont
	if span.Font != nil {
		font = span.Font
	}
	for _, r := range text {
		bounds, advance, ok := font.face.GlyphBounds(r)
		if !ok {
			return nil, fmt.Errorf("获取文字度量失败")
		}
		runes.PushBack(textRune{
			r:    r,
			font: font,
			metrics: textRuneMetrics{
				ascent:  -bounds.Min.Y,
				descent: bounds.Max.Y,
				advance: advance,
			},
		})
	}
	return runes, nil
}
