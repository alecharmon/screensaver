package screensaver

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	starCount = 100
	starSpeed = 0.03
	starMinZ  = 0.1
	starMaxZ  = 3.0
	starTick  = 33 * time.Millisecond
)

const (
	logoShineInterval = 10 * time.Second
	logoShineDelay    = 2 * time.Second
	logoShineTickRate = 50 * time.Millisecond
	logoShineStep     = 2
	logoShineBand     = 4
	quoteShineDelay   = 2 * time.Second
	quoteShineTick    = 60 * time.Millisecond
	quoteShineStep    = 2
	quoteShineBand    = 8
)

var (
	starBrightStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	starDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	logoBaseStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	logoShineStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	quoteBaseStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	quoteShineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
)

var logoArt = []string{
	`          ████                `,
	`        ████████              `,
	`      ███████████             `,
	`    ███████████████      ████ `,
	`      ███████████████████████ `,
	`        ███████████████████   `,
	`          ███████████████     `,
	`            ███████████       `,
	`          ███████             `,
	`        █████                 `,
}

var (
	leftDots  = [4]rune{0x40, 0x04, 0x02, 0x01}
	rightDots = [4]rune{0x80, 0x20, 0x10, 0x08}
)

type starfieldTickMsg struct{}
type logoShineStartMsg struct{}
type logoShineStepMsg struct{}
type quoteShineStartMsg struct{}
type quoteShineTickMsg struct{}
type quoteLoadedMsg struct {
	Quote Quote
	Err   error
}

type star struct {
	x, y, z float64
}

type starCell struct {
	ch     rune
	bright bool
}

type starfield struct {
	width, height int
	stars         []star
	rng           *rand.Rand
	grid          [][]starCell
}

type logo struct {
	lines    [][]rune
	shinePos int
	maxDiag  int
}

type Model struct {
	width, height int
	starfield     *starfield
	quote         Quote
	quoteErr      error
	quoteLoaded   bool
	quoteShinePos int
	quoteShineOn  bool
	nextQuoteCmd  func() tea.Cmd
}

func New() Model {
	return Model{
		starfield:    newStarfield(),
		nextQuoteCmd: loadRandomQuoteCmd,
	}
}

func Run() error {
	_, err := tea.NewProgram(New(), tea.WithAltScreen()).Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.starfield.init(), loadQuoteCmd(time.Now()), scheduleQuoteShineStart())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.quoteLoaded && m.quote.Text != "" && !m.canShowQuote() {
			return m, tea.Quit
		}
		return m, m.starfield.update(msg)
	case tea.KeyMsg, tea.MouseMsg:
		if key, ok := msg.(tea.KeyMsg); ok && key.String() == "n" {
			return m, m.nextQuoteCmd()
		}
		return m, tea.Quit
	case quoteLoadedMsg:
		m.quote = msg.Quote
		m.quoteErr = msg.Err
		m.quoteLoaded = true
		if m.width > 0 && m.height > 0 && m.quote.Text != "" && !m.canShowQuote() {
			return m, tea.Quit
		}
		return m, nil
	case quoteShineStartMsg:
		m.quoteShineOn = true
		return m, scheduleQuoteShineTick()
	case quoteShineTickMsg:
		m.quoteShinePos += quoteShineStep
		if m.quoteShinePos > 200 {
			m.quoteShinePos = 0
		}
		return m, scheduleQuoteShineTick()
	case starfieldTickMsg:
		return m, m.starfield.update(msg)
	}
	return m, nil
}

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	m.starfield.computeGrid()
	panel := m.quotePanel()
	panelLines := strings.Split(panel, "\n")
	panelWidth, panelHeight := panelDimensions(panel)
	left := centeredStart(m.width, panelWidth)
	top := centeredStart(m.height, panelHeight)

	lines := make([]string, 0, m.height)
	for row := 0; row < m.height; row++ {
		if row >= top && row < top+panelHeight {
			panelLine := panelLines[row-top]
			if left >= m.width {
				lines = append(lines, m.starfield.renderFullRow(row))
				continue
			}
			maxLineWidth := m.width - left
			panelLine = ansi.Truncate(panelLine, maxLineWidth, "")
			lineWidth := ansi.StringWidth(panelLine)
			rightStart := left + lineWidth

			line := m.starfield.renderRow(row, 0, left) + ansi.ResetStyle + panelLine + ansi.ResetStyle + m.starfield.renderRow(row, rightStart, m.width)
			lines = append(lines, line)
			continue
		}
		lines = append(lines, m.starfield.renderFullRow(row))
	}

	return strings.Join(lines, "\n")
}

func (m Model) quotePanel() string {
	const panelWidth = 64
	style := lipgloss.NewStyle().
		Width(panelWidth).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("8")).
		Padding(1, 2).
		Foreground(lipgloss.Color("15"))

	if !m.quoteLoaded {
		return style.Render("Loading quote of the day...")
	}

	if m.quote.Text == "" {
		if m.quoteErr != nil {
			return style.Render("Could not load quote.\nSet ZENQUOTES_API_KEY and restart.")
		}
		return style.Render("No quote available.")
	}

	author := m.quote.Author
	if author == "" {
		author = "Unknown"
	}
	content := wrapText(m.quote.Text, panelWidth-6) + "\n\n- " + author
	content = renderShinyQuote(content, m.quoteShinePos, m.quoteShineOn)
	return style.Render(content)
}

func (m Model) canShowQuote() bool {
	panel := m.quotePanel()
	panelWidth, panelHeight := panelDimensions(panel)
	return panelWidth <= m.width && panelHeight <= m.height
}

func panelDimensions(panel string) (int, int) {
	lines := strings.Split(panel, "\n")
	width := 0
	for _, line := range lines {
		if w := ansi.StringWidth(line); w > width {
			width = w
		}
	}
	return width, len(lines)
}

func wrapText(s string, width int) string {
	if width <= 0 {
		return s
	}
	words := strings.Fields(s)
	if len(words) == 0 {
		return ""
	}

	lines := []string{words[0]}
	for _, w := range words[1:] {
		cur := lines[len(lines)-1]
		if len(cur)+1+len(w) <= width {
			lines[len(lines)-1] = cur + " " + w
			continue
		}
		lines = append(lines, w)
	}
	return strings.Join(lines, "\n")
}

func renderShinyQuote(content string, shinePos int, shineOn bool) string {
	lines := strings.Split(content, "\n")
	for row, line := range lines {
		runes := []rune(line)
		var sb strings.Builder
		for col, r := range runes {
			idx := row + col
			if shineOn && idx >= shinePos && idx < shinePos+quoteShineBand {
				sb.WriteString(quoteShineStyle.Render(string(r)))
			} else {
				sb.WriteString(quoteBaseStyle.Render(string(r)))
			}
		}
		lines[row] = sb.String()
	}
	return strings.Join(lines, "\n")
}

func scheduleQuoteShineTick() tea.Cmd {
	return tea.Tick(quoteShineTick, func(time.Time) tea.Msg {
		return quoteShineTickMsg{}
	})
}

func scheduleQuoteShineStart() tea.Cmd {
	return tea.Tick(quoteShineDelay, func(time.Time) tea.Msg {
		return quoteShineStartMsg{}
	})
}

func centeredStart(total, content int) int {
	if total <= content {
		return 0
	}
	return (total - content) / 2
}

func logoDimensions() (int, int) {
	maxWidth := 0
	for _, line := range logoArt {
		w := len([]rune(line))
		if w > maxWidth {
			maxWidth = w
		}
	}
	return maxWidth, len(logoArt)
}

func newStarfield() *starfield {
	return &starfield{
		rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

func (s *starfield) init() tea.Cmd {
	return s.scheduleTick()
}

func (s *starfield) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.resize(msg.Width, msg.Height)
	case starfieldTickMsg:
		s.step()
		return s.scheduleTick()
	}
	return nil
}

func (s *starfield) renderRow(row, fromCol, toCol int) string {
	if row < 0 || row >= s.height {
		return strings.Repeat(" ", max(toCol-fromCol, 0))
	}

	var sb strings.Builder
	for col := fromCol; col < toCol; col++ {
		if col < 0 || col >= s.width {
			sb.WriteByte(' ')
			continue
		}
		cell := &s.grid[row][col]
		if cell.ch == 0 {
			sb.WriteByte(' ')
			continue
		}
		if cell.bright {
			sb.WriteString(starBrightStyle.Render(string(cell.ch)))
			continue
		}
		sb.WriteString(starDimStyle.Render(string(cell.ch)))
	}
	return sb.String()
}

func (s *starfield) renderFullRow(row int) string {
	return s.renderRow(row, 0, s.width)
}

func (s *starfield) computeGrid() {
	if s.width <= 0 || s.height <= 0 {
		return
	}

	for row := range s.height {
		for col := range s.width {
			s.grid[row][col] = starCell{}
		}
	}

	subW := s.width * 2
	subH := s.height * 4
	centerX := float64(subW) / 2
	centerY := float64(subH) / 2

	for i := range s.stars {
		st := &s.stars[i]
		if st.z <= 0 {
			continue
		}

		sx := centerX + st.x/st.z
		sy := centerY + st.y/st.z

		sxi := int(sx)
		syi := int(sy)
		if sxi < 0 || sxi >= subW || syi < 0 || syi >= subH {
			continue
		}

		col := sxi / 2
		row := syi / 4
		dotCol := sxi % 2
		dotRow := syi % 4
		dotIndex := 3 - dotRow

		cell := &s.grid[row][col]
		if cell.ch == 0 {
			cell.ch = 0x2800
		}
		if dotCol == 0 {
			cell.ch |= leftDots[dotIndex]
		} else {
			cell.ch |= rightDots[dotIndex]
		}
		if st.z < starMaxZ/2 {
			cell.bright = true
		}
	}
}

func (s *starfield) resize(width, height int) {
	s.width = width
	s.height = height
	s.stars = make([]star, starCount)
	for i := range s.stars {
		s.stars[i] = s.randomStar()
	}
	s.grid = make([][]starCell, height)
	for row := range height {
		s.grid[row] = make([]starCell, width)
	}
}

func (s *starfield) step() {
	subW := s.width * 2
	subH := s.height * 4
	centerX := float64(subW) / 2
	centerY := float64(subH) / 2

	for i := range s.stars {
		st := &s.stars[i]
		st.z -= starSpeed

		if st.z <= starMinZ {
			s.stars[i] = s.randomStar()
			continue
		}

		sx := centerX + st.x/st.z
		sy := centerY + st.y/st.z
		if sx < 0 || sx >= float64(subW) || sy < 0 || sy >= float64(subH) {
			s.stars[i] = s.randomStar()
		}
	}
}

func (s *starfield) randomStar() star {
	spread := float64(max(s.width, s.height))
	return star{
		x: (s.rng.Float64() - 0.5) * spread,
		y: (s.rng.Float64() - 0.5) * spread,
		z: starMinZ + s.rng.Float64()*(starMaxZ-starMinZ),
	}
}

func (s *starfield) scheduleTick() tea.Cmd {
	return tea.Tick(starTick, func(time.Time) tea.Msg {
		return starfieldTickMsg{}
	})
}

func newLogo() *logo {
	lines := make([][]rune, len(logoArt))
	maxWidth := 0
	for i, line := range logoArt {
		lines[i] = []rune(line)
		if len(lines[i]) > maxWidth {
			maxWidth = len(lines[i])
		}
	}

	return &logo{
		lines:    lines,
		shinePos: -1,
		maxDiag:  maxWidth + len(logoArt),
	}
}

func (l *logo) init() tea.Cmd {
	return tea.Tick(logoShineDelay, func(time.Time) tea.Msg {
		return logoShineStartMsg{}
	})
}

func (l *logo) update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case logoShineStartMsg:
		l.shinePos = 0
		return l.shineTick()
	case logoShineStepMsg:
		l.shinePos += logoShineStep
		if l.shinePos > l.maxDiag+logoShineBand {
			l.shinePos = -1
			return tea.Tick(logoShineInterval, func(time.Time) tea.Msg {
				return logoShineStartMsg{}
			})
		}
		return l.shineTick()
	}
	return nil
}

func (l *logo) viewLine(row int) string {
	return l.renderLine(l.lines[row], row)
}

func (l *logo) renderLine(line []rune, row int) string {
	if l.shinePos < 0 {
		return logoBaseStyle.Render(string(line))
	}

	shineStart := l.shinePos - row
	shineEnd := shineStart + logoShineBand

	lineLen := len(line)
	if shineStart >= lineLen || shineEnd <= 0 {
		return logoBaseStyle.Render(string(line))
	}

	shineStart = max(shineStart, 0)
	shineEnd = min(shineEnd, lineLen)

	var sb strings.Builder
	if shineStart > 0 {
		sb.WriteString(logoBaseStyle.Render(string(line[:shineStart])))
	}
	sb.WriteString(logoShineStyle.Render(string(line[shineStart:shineEnd])))
	if shineEnd < lineLen {
		sb.WriteString(logoBaseStyle.Render(string(line[shineEnd:])))
	}
	return sb.String()
}

func (l *logo) shineTick() tea.Cmd {
	return tea.Tick(logoShineTickRate, func(time.Time) tea.Msg {
		return logoShineStepMsg{}
	})
}
