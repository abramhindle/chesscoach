// Unnamed Chess Coach Program

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"log"
	"os/exec"

	"github.com/golang/freetype/truetype"
	eb "github.com/hajimehoshi/ebiten"
	ebu "github.com/hajimehoshi/ebiten/ebitenutil"
	ebf "github.com/hajimehoshi/ebiten/examples/resources/fonts"
	ebi "github.com/hajimehoshi/ebiten/inpututil"
	ebt "github.com/hajimehoshi/ebiten/text"
	"github.com/notnil/chess"
	"golang.org/x/image/font"
	uci "gopkg.in/freeeve/uci.v1"
)

const cellSize = 48 // size of grid cells
const textX = cellSize*8 + 16

var (
	// -- state of the current game
	game *chess.Game
	// store the moves in algebraic notation as we go.
	// it's inconvenient to calculate them from just the list of move objects
	// at the end, as they require building the position at each step
	moves []string
	bestMove string
	eng   *uci.Engine
	// -- ui state --
	p0 = chess.NoSquare // player's selected square
	// -- assets ---
	mainFont font.Face
	icons    [13]*eb.Image // 13 = card(piece)
)

func sprite(path string) *eb.Image {
	im, _, err := ebu.NewImageFromFile(path, eb.FilterDefault)
	if err != nil {
		log.Fatal(err)
	}
	return im
}

func getEngine() *uci.Engine {
	eng, err := uci.NewEngine("./stockfish")
	if err != nil {
		log.Println(err)
		return nil
	}
	eng.SetOptions(uci.Options{
		Hash:    128,
		Ponder:  false,
		OwnBook: true,
		MultiPV: 32,
	})
	return eng
}

func init() {
	game = chess.NewGame(chess.UseNotation(chess.AlgebraicNotation{}))
	eng = getEngine()
	updateEngine()
	icons[chess.WhitePawn] = sprite("sprites/wp.png")
	icons[chess.WhiteRook] = sprite("sprites/wr.png")
	icons[chess.WhiteKnight] = sprite("sprites/wn.png")
	icons[chess.WhiteBishop] = sprite("sprites/wb.png")
	icons[chess.WhiteQueen] = sprite("sprites/wq.png")
	icons[chess.WhiteKing] = sprite("sprites/wk.png")
	icons[chess.BlackPawn] = sprite("sprites/bp.png")
	icons[chess.BlackRook] = sprite("sprites/br.png")
	icons[chess.BlackKnight] = sprite("sprites/bn.png")
	icons[chess.BlackBishop] = sprite("sprites/bb.png")
	icons[chess.BlackQueen] = sprite("sprites/bq.png")
	icons[chess.BlackKing] = sprite("sprites/bk.png")

	// set up the font
	tt, err := truetype.Parse(ebf.ArcadeN_ttf)
	if err != nil {
		log.Fatal(err)
	}
	mainFont = truetype.NewFace(tt, &truetype.Options{
		Size:    8,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}

func blit(screen, im *eb.Image, gx, gy int) {
	op := &eb.DrawImageOptions{}
	op.GeoM.Translate(float64(gx*cellSize), float64(gy*cellSize))
	screen.DrawImage(im, op)
}

func squareAt(x, y int) chess.Square {
	if x < 0 || x > 7 || y < 0 || y > 7 {
		return chess.NoSquare
	}
	return chess.Square(int(8*(7-y)) + x)
}

func mouseSquare() chess.Square {
	x, y := eb.CursorPosition()
	if x < 0 || y < 0 {
		return chess.NoSquare
	}
	x, y = x/cellSize, y/cellSize
	return squareAt(x, y)
}

func isValidSquare(sq chess.Square) bool {
	for _, mv := range game.ValidMoves() {
		if mv.S1() == sq {
			return true
		}
	}
	return false
}

func validSquares() (result []chess.Square) {
	counts := make(map[chess.Square]int)
	for _, mv := range game.ValidMoves() {
		if p0 == chess.NoSquare {
			counts[mv.S1()]++
		} else if p0 == mv.S1() {
			counts[mv.S2()]++
		}
	}
	for sq := range counts {
		result = append(result, sq)
	}
	return result
}

// Checks whether the start and end squares comprise a valid move.
// TODO: pawn promotion.
func validMove(sq0, sq1 chess.Square) (valid bool, mv *chess.Move) {
	for _, mv = range game.ValidMoves() {
		if mv.S1() == sq0 && mv.S2() == sq1 {
			return true, mv
		}
	}
	return false, nil
}

func updateEngine() {
	fmt.Println("game:", game)
	fmt.Println("turn:", game.Position().Turn())
	fmt.Println("FEN:", game.Position())

	if eng != nil {
		eng.SetFEN(game.Position().String())
		// set some result filter options
		resultOpts := uci.HighestDepthOnly | uci.IncludeUpperbounds | uci.IncludeLowerbounds
		results, _ := eng.GoDepth(10, resultOpts)

		// print it (String() goes to pretty JSON for now)
		for _, r := range results.Results {
				fmt.Println(r.BestMoves[0], r.Score)
		}
		fmt.Println("bestmove: ", results.BestMove)
		bestMove = results.BestMove
		if game.Position().Turn() == chess.Black {
			addMoveStr(results.BestMove)
		}
	}
}

func addMoveStr(s string) {
	var note chess.LongAlgebraicNotation
	mv, err := note.Decode(game.Position(), s)
	if err != nil {
		log.Println(err)
	} else {
		addMove(mv)
	}
}

func addMove(mv *chess.Move) {
	alg := chess.AlgebraicNotation{}
	moves = append(moves, alg.Encode(game.Position(), mv))
	err := game.Move(mv)
	if err == nil {
		p0 = chess.NoSquare
	} else {
		moves = moves[:len(moves)-1]
		log.Println(err)
	}
	updateEngine()
}
func myRun(prog string, arg string) {
	cmd := exec.Command(prog, arg)
	err := cmd.Run()
	if err != nil {
	    log.Fatal(err)
	}
}
func mySuccess(arg string) {
	myRun("./success.sh",arg)
}
func myFailure(arg string) {
	myRun("./failure.sh",arg)
}
func myInvalid() {
	myRun("./invalid.sh","")
}
func watchMouse() {
	if ebi.IsMouseButtonJustPressed(eb.MouseButtonLeft) {
		ms := mouseSquare()
		switch {
		case p0 == chess.NoSquare && isValidSquare(ms):
			p0 = ms
		case p0 == ms: // click again to disable
			p0 = chess.NoSquare
		default:
			if valid, mv := validMove(p0, ms); valid {
				if mv.String() == bestMove {
					fmt.Println("Adding moving", mv)
					mySuccess(mv.String())
					addMove(mv)
				} else {
					fmt.Println("Not best move: ", mv)
					myFailure(mv.String())
				}
			} else {
				fmt.Println("Not valid: ", mv, p0, ms)
				myInvalid()
			}
		}
	}
}

func drawSquareXY(screen *eb.Image, x, y int, c color.Color) {
	x0, y0 := x*cellSize, (7-y)*cellSize
	rect := image.Rect(x0, y0, x0+cellSize-1, y0+cellSize-1)
	draw.Draw(screen, rect, &image.Uniform{c}, image.ZP, draw.Src)
}

func drawSquare(screen *eb.Image, sq chess.Square, c color.Color) {
	drawSquareXY(screen, int(sq.File()), int(sq.Rank()), c)
}

func drawBoard(screen *eb.Image) {
	light := color.RGBA{0, 0, 255, 255}
	dark := color.RGBA{0, 0, 127, 255}
	var c color.Color
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			if (y & 1) == (x & 1) {
				c = &light
			} else {
				c = &dark
			}
			drawSquareXY(screen, x, 7-y, c)
		}
	}
}

func drawMarks(screen *eb.Image) {
	highlight := color.RGBA{224, 164, 0, 63}
	validLight := color.RGBA{63, 63, 127, 63}
	validDark := color.RGBA{32, 32, 64, 63}
	for _, sq := range validSquares() {
		if int(sq.Rank())&1 == int(sq.File())&1 {
			drawSquare(screen, sq, validDark)
		} else {
			drawSquare(screen, sq, validLight)
		}
	}
	if p0 != chess.NoSquare {
		drawSquare(screen, p0, highlight)
	}
}

func drawPieces(screen *eb.Image) {
	for sq, p := range game.Position().Board().SquareMap() {
		blit(screen, icons[p], int(sq.File()), 7-int(sq.Rank()))
	}
}

func drawText(screen *eb.Image) {
	ebt.Draw(screen, "Hello. I am chess coach.", mainFont, textX, 16, color.White)
	ebt.Draw(screen, "You are playing white today.", mainFont, textX, 32, color.White)
	ebt.Draw(screen, "Use mouse to select move.", mainFont, textX, 48, color.White)
	if ms := mouseSquare(); ms != chess.NoSquare {
		ebt.Draw(screen, fmt.Sprintf("Mouse is over %s", ms), mainFont, textX, 64, color.White)
	}
}

func drawMoves(screen *eb.Image) {
	for i := 0; i < len(moves); i += 2 {
		n := i >> 1
		w, b := moves[i], "*"
		if i+1 < len(moves) {
			b = moves[i+1]
		}
		ebt.Draw(screen, fmt.Sprintf("%d. %s %s", n+1, w, b), mainFont, textX, 96+16*n, color.White)
	}
}

func update(screen *eb.Image) error {
	watchMouse()
	if !eb.IsDrawingSkipped() {
		drawBoard(screen)
		drawMarks(screen)
		drawPieces(screen)
		drawText(screen)
		drawMoves(screen)
	}
	return nil
}

func main() {
	if err := eb.Run(update, 640, cellSize*8, 2, "chesscoach"); err != nil {
		log.Fatal(err)
	}
}
