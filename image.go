package psbotfunc

import (
	"image"
	"image/color"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"cloud.google.com/go/vision/v2/apiv1/visionpb"
	"go.uber.org/zap"
)

type Image struct {
	Logger *zap.Logger `json:"-"`
	Format string
	Bytes  io.Reader

	Width  int
	Height int

	DetectedTexts []*DetectedText
	DetectedFoods map[string]int
}

func NewImage(imageBytes io.Reader, filetype string, width, height int, logger *zap.Logger) (*Image, error) {
	return &Image{
		Logger: logger,
		Bytes:  imageBytes,
		Format: filetype,
		Width:  width,
		Height: height,
	}, nil
}

func (i *Image) UpdateImage(annotations []*visionpb.EntityAnnotation) error {
	i.DetectedFoods = make(map[string]int)

	dtexts := []*DetectedText{}
	for id, annotation := range annotations {
		dtext := NewDetectedText(strconv.Itoa(id), annotation, i.Width, i.Height, i.Logger)
		dtexts = append(dtexts, dtext)
	}
	i.DetectedTexts = dtexts

	i.MarshalDetectedTexts()
	return nil
}

func (i *Image) MarshalDetectedTexts() error {
	points := []DetectedText{}
	for _, dtext := range i.DetectedTexts {
		points = append(points, *dtext)
	}
	clusters := Clusterize(points[1:], 1, 0.01, i.Logger)
	merged := []*DetectedText{}
	for _, cluster := range clusters {
		m := Merge(cluster...)
		merged = append(merged, m)
	}
	i.DetectedTexts = merged
	return nil
}

func (i *Image) DetetcFoods(foods []*Food) {
	for _, dtext := range i.DetectedTexts {
		if isFood, food := dtext.IsFood(foods); isFood {
			i.DetectedFoods[food.Name] = GetFoodNum(dtext, i.DetectedTexts)
			i.Logger.Debug(food.Name, zap.Int("num", i.DetectedFoods[food.Name]))
		}
	}
}

func (i *Image) GetCookResult(cooks []*Cook) (string, string) {
	var makables string
	var unmakables string
	for _, cook := range cooks {
		if i.isMakable(cook) {
			makables += "    " + cook.Name + "\n"
			for _, food := range cook.Recipe {
				makables += "        " + food.Name + " x" + strconv.Itoa(food.Num) + "\n"
			}
		} else {
			unmakables += "    " + cook.Name + "\n"
			for _, food := range cook.Recipe {
				var shortage int
				if v, ok := i.DetectedFoods[food.Name]; ok {
					shortage = food.Num - v
				} else {
					shortage = food.Num
				}
				if shortage <= 0 {
					unmakables += "        " + food.Name + " x" + strconv.Itoa(food.Num) + "\n"
				} else {
					unmakables += "        " + food.Name + " x" + strconv.Itoa(food.Num) + " あと" + strconv.Itoa(shortage) + "\n"
				}
			}
		}
	}
	return "作れるレシピ:\n" + makables, "作れないレシピ:\n" + unmakables
}

func (i *Image) isMakable(cook *Cook) bool {
	for _, food := range cook.Recipe {
		num, ok := i.DetectedFoods[food.Name]
		if !ok {
			return false
		} else if num < food.Num {
			return false
		}
	}
	return true
}

type TextType int

const (
	FOODS = iota
	NUM
	OTHER
)

type DetectedText struct {
	Logger *zap.Logger `json:"-"`

	ID   string   `json:"id"`
	Text []string `json:"text"`
	Type TextType

	MinX int `json:"-"`
	MinY int `json:"-"`
	MaxX int `json:"-"`
	MaxY int `json:"-"`

	NMinX float32 `json:"-"`
	NMinY float32 `json:"-"`
	NMaxX float32 `json:"-"`
	NMaxY float32 `json:"-"`
}

func NewDetectedText(id string, a *visionpb.EntityAnnotation, w, h int, logger *zap.Logger) *DetectedText {
	var x1, x2, y1, y2 = w, 0, h, 0
	for _, vertex := range a.GetBoundingPoly().GetVertices() {
		x1 = int(math.Min(float64(x1), float64(vertex.GetX())))
		x2 = int(math.Max(float64(x2), float64(vertex.GetX())))
		y1 = int(math.Min(float64(y1), float64(vertex.GetY())))
		y2 = int(math.Max(float64(y2), float64(vertex.GetY())))
	}
	return &DetectedText{
		Logger: logger,
		ID:     id,
		Text:   []string{a.Description},
		MinX:   x1,
		MinY:   y1,
		MaxX:   x2,
		MaxY:   y2,
		NMinX:  float32(x1) / float32(w),
		NMinY:  float32(y1) / float32(w),
		NMaxX:  float32(x2) / float32(w),
		NMaxY:  float32(y2) / float32(w),
	}
}

func (d DetectedText) GetID() string {
	return d.ID
}

func (d DetectedText) Distance(other DetectedText) float64 {
	return d.NDistanceFrom(other)
}

func (d DetectedText) DistanceFrom(other DetectedText) float64 {
	var dx, dy float64
	if (d.MinX <= other.MinX && other.MinX <= d.MaxX) ||
		(d.MinX <= other.MaxX && other.MaxX <= d.MaxX) {
		dx = 0
	} else {
		dx = math.Min(math.Abs(float64(d.MinX-other.MaxX)), math.Abs(float64(d.MaxX-other.MinX)))
	}
	if (d.MinY <= other.MinY && other.MinY <= d.MaxY) ||
		(d.MinY <= other.MaxY && other.MaxY <= d.MaxY) {
		dy = 0
	} else {
		dy = math.Min(math.Abs(float64(d.MinY-other.MaxY)), math.Abs(float64(d.MaxY-other.MinY)))
	}
	dist := math.Sqrt(dx*dx + dy*dy)
	return dist
}

func (d DetectedText) NDistanceFrom(other DetectedText) float64 {
	var dx, dy float64
	if (d.NMinX <= other.NMinX && other.NMinX <= d.NMaxX) ||
		(d.NMinX <= other.NMaxX && other.NMaxX <= d.NMaxX) ||
		(other.NMinX <= d.NMinX && d.NMinX <= other.NMaxX) ||
		(other.NMinX <= d.NMaxX && d.NMaxX <= other.NMaxX) {
		dx = 0
	} else {
		dx = math.Min(math.Abs(float64(d.NMinX-other.NMaxX)), math.Abs(float64(d.NMaxX-other.NMinX)))
	}
	if (d.NMinY <= other.NMinY && other.NMinY <= d.NMaxY) ||
		(d.NMinY <= other.NMaxY && other.NMaxY <= d.NMaxY) ||
		(other.NMinY <= d.NMinY && d.NMinY <= other.NMaxY) ||
		(other.NMinY <= d.NMaxY && d.NMaxY <= other.NMaxY) {
		dy = 0
	} else {
		dy = math.Min(math.Abs(float64(d.NMinY-other.NMaxY)), math.Abs(float64(d.NMaxY-other.NMinY)))
	}
	dist := math.Sqrt(dx*dx + dy*dy)
	// d.Logger.Info("distance", zap.Any(d.ID, d.Text), zap.Any(other.ID, other.Text), zap.Float64("dist", dist), zap.Float64("x", dx), zap.Float64("y", dy))
	return dist
}

func (d *DetectedText) IsFood(foods []*Food) (bool, *Food) {
	acc := make(map[*Food]float64)
	for _, food := range foods {
		chars := len(food.Name)
		match := 0
		ans := food.Name
		for _, text := range d.Text {
			if strings.Contains(ans, text) {
				match += len(text)
				ans = strings.Replace(ans, text, "", 1)
			} else {
				match = int(math.Max(float64(match-len(text)), float64(0)))
			}
		}
		match = int(math.Max(float64(match-len(ans)), float64(0)))
		accuracy := float64(match) / float64(chars)
		if accuracy > 0.5 {
			acc[food] = accuracy
		}
	}

	if len(acc) == 0 {
		return false, nil
	}

	var max float64
	var maxfood *Food
	for k, v := range acc {
		if max < v {
			max = v
			maxfood = k
		}
	}
	return true, maxfood
}

var numPattern = regexp.MustCompile(`x([0-9]+)`)

func GetFoodNum(foodtext *DetectedText, dtexts []*DetectedText) int {
	minDist := foodtext.Distance(*dtexts[0])
	numText := dtexts[0]
	for _, dtext := range dtexts {
		if numPattern.Match([]byte(dtext.Text[0])) {
			dist := foodtext.Distance(*dtext)
			if dist < minDist {
				numText = dtext
				minDist = dist
			}
		}
	}

	num, err := strconv.Atoi(numText.Text[0][1:])
	if err != nil {
		return 0
	}
	return num
}

func (d *DetectedText) DrawRect(canvas *image.RGBA) {
	lineColor := color.RGBA{R: 255, G: 0, B: 0, A: 128}
	for i := d.MinX; i < d.MaxX; i++ {
		canvas.Set(i, d.MinY, lineColor)
		canvas.Set(i, d.MinY-1, lineColor)
		canvas.Set(i, d.MinY+1, lineColor)
		canvas.Set(i, d.MaxY, lineColor)
		canvas.Set(i, d.MaxY, lineColor)
		canvas.Set(i, d.MaxY, lineColor)
	}

	for i := d.MinY; i < d.MaxY; i++ {
		canvas.Set(d.MinX, i, lineColor)
		canvas.Set(d.MinX-1, i, lineColor)
		canvas.Set(d.MinX+1, i, lineColor)
		canvas.Set(d.MaxX, i, lineColor)
		canvas.Set(d.MaxX, i, lineColor)
		canvas.Set(d.MaxX, i, lineColor)
	}
}

func Merge(dtexts ...DetectedText) *DetectedText {
	ret := dtexts[0]
	for i := 1; i < len(dtexts); i++ {
		dtext := dtexts[i]
		ret.Text = append(ret.Text, dtext.Text...)
		ret.MinX = int(math.Min(float64(ret.MinX), float64(dtext.MinX)))
		ret.MaxX = int(math.Max(float64(ret.MaxX), float64(dtext.MaxX)))
		ret.MinY = int(math.Min(float64(ret.MinY), float64(dtext.MinY)))
		ret.MaxY = int(math.Max(float64(ret.MaxY), float64(dtext.MaxY)))
		ret.NMinX = float32(math.Min(float64(ret.NMinX), float64(dtext.NMinX)))
		ret.NMaxX = float32(math.Max(float64(ret.NMaxX), float64(dtext.NMaxX)))
		ret.NMinY = float32(math.Min(float64(ret.NMinY), float64(dtext.NMinY)))
		ret.NMaxY = float32(math.Max(float64(ret.NMaxY), float64(dtext.NMaxY)))
	}
	return &ret
}

func In(word string, words []string) bool {
	for _, w := range words {
		if w == word {
			return true
		}
	}
	return false
}
