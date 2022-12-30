package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

type Component int

const (
	TobaccoComponent Component = iota + 1
	PapersComponent
	LighterComponent
)

func (c Component) String() string {
	switch c {
	case TobaccoComponent:
		return "tobacco"
	case PapersComponent:
		return "papers"
	case LighterComponent:
		return "lighter"
	default:
		return "unknown"
	}
}

type Components []Component

func (arr Components) String() string {
	var str []string
	for _, c := range arr {
		str = append(str, c.String())
	}

	front := str[:len(str)-1]

	return fmt.Sprintf("%s and %s", strings.Join(front, ", "), str[len(str)-1])
}

type Tobacco struct{}
type Papers struct{}
type Lighter struct{}

type AvailableComponents struct {
	Tobacco <-chan Tobacco
	Papers  <-chan Papers
	Lighter <-chan Lighter
}

type SmokerWithTobacco struct {
	Papers  <-chan Papers
	Lighter <-chan Lighter
}

type SmokerWithPapers struct {
	Tobacco <-chan Tobacco
	Lighter <-chan Lighter
}

type SmokerWithLighter struct {
	Tobacco <-chan Tobacco
	Papers  <-chan Papers
}

func getComponents() Components {
	var (
		forSWT = Components{
			LighterComponent,
			PapersComponent,
		}

		forSWP = Components{
			LighterComponent,
			TobaccoComponent,
		}

		forSWL = Components{
			TobaccoComponent,
			PapersComponent,
		}

		arr = []Components{
			forSWT, forSWL, forSWP,
		}
	)

	rand.Seed(time.Now().Unix())

	index := rand.Intn(3)

	return arr[index]
}

func agent(
	stop <-chan struct{}, signal <-chan struct{},
) (<-chan struct{}, AvailableComponents) {

	var (
		done      = make(chan struct{})
		tobaccoCh = make(chan Tobacco)
		papersCh  = make(chan Papers)
		lighterCh = make(chan Lighter)

		ac = AvailableComponents{
			Tobacco: tobaccoCh,
			Papers:  papersCh,
			Lighter: lighterCh,
		}
	)

	cleanup := func() {
		defer close(done)

		close(tobaccoCh)
		close(papersCh)
		close(lighterCh)
		log.Println("agent: shutdown complete")
	}

	go func() {
		defer cleanup()

		for {
			select {
			case <-stop:
				return
			case <-signal:
				components := getComponents()

				log.Printf("agent: sending %v\n", components)

				for _, c := range components {
					switch c {
					case TobaccoComponent:
						select {
						case <-stop:
							return
						case tobaccoCh <- Tobacco{}:
						}
					case PapersComponent:
						select {
						case <-stop:
							return
						case papersCh <- Papers{}:
						}
					case LighterComponent:
						select {
						case <-stop:
							return
						case lighterCh <- Lighter{}:
						}
					default:
						log.Fatalf("agent: received unknown value %d", c)
						return
					}
				}
			}
		}
	}()

	return done, ac

}

func table(
	stop <-chan struct{}, ac AvailableComponents,
) (<-chan struct{}, SmokerWithTobacco, SmokerWithPapers, SmokerWithLighter) {

	var (
		done = make(chan struct{})

		swtPapersCh  = make(chan Papers)
		swtLighterCh = make(chan Lighter)
		swt          = SmokerWithTobacco{
			Papers:  swtPapersCh,
			Lighter: swtLighterCh,
		}

		swpTobaccoCh = make(chan Tobacco)
		swpLighterCh = make(chan Lighter)
		swp          = SmokerWithPapers{
			Tobacco: swpTobaccoCh,
			Lighter: swpLighterCh,
		}

		swlTobaccoCh = make(chan Tobacco)
		swlPapersCh  = make(chan Papers)
		swl          = SmokerWithLighter{
			Tobacco: swlTobaccoCh,
			Papers:  swlPapersCh,
		}
	)

	cleanup := func() {
		defer close(done)

		log.Println("table: shutdown complete")
	}

	go func() {
		defer cleanup()

		for {

			var (
				hasPapers  bool
				hasLighter bool
				hasTobacco bool
			)

			for components := 0; components < 2; components++ {
				select {
				case <-stop:
					return
				case <-ac.Tobacco:
					hasTobacco = true
				case <-ac.Papers:
					hasPapers = true
				case <-ac.Lighter:
					hasLighter = true
				}
			}

			if hasPapers && hasLighter {
				select {
				case <-stop:
					return
				case swtPapersCh <- Papers{}:
				}

				select {
				case <-stop:
					return
				case swtLighterCh <- Lighter{}:
				}

				continue
			}

			if hasTobacco && hasLighter {
				select {
				case <-stop:
					return
				case swpTobaccoCh <- Tobacco{}:
				}

				select {
				case <-stop:
					return
				case swpLighterCh <- Lighter{}:
				}

				continue
			}

			if hasPapers && hasTobacco {
				select {
				case <-stop:
					return
				case swlPapersCh <- Papers{}:
				}

				select {
				case <-stop:
					return
				case swlTobaccoCh <- Tobacco{}:
				}
			}

		}

	}()

	return done, swt, swp, swl
}

func hasLighter(
	stop <-chan struct{},
	tobaccoCh <-chan Tobacco,
	papersCh <-chan Papers,
) (<-chan struct{}, <-chan string) {

	var (
		done     = make(chan struct{})
		messages = make(chan string)
	)

	cleanup := func() {
		defer close(done)

		close(messages)
		log.Println("hasLighter: shutdown complete")
	}

	go func() {
		defer cleanup()

		for {
			var hasPapers bool
			var hasTobacco bool

			for !(hasPapers && hasTobacco) {
				select {
				case <-stop:
					return
				case <-papersCh:
					hasPapers = true
				case <-tobaccoCh:
					hasTobacco = true
				}
			}

			msg := fmt.Sprintf("hasLighter: rolled and smoked a cigarette")

			select {
			case <-stop:
				return
			case messages <- msg:
			}
		}
	}()

	return done, messages
}

func hasPapers(
	stop <-chan struct{},
	tobaccoCh <-chan Tobacco,
	lighterCh <-chan Lighter,
) (<-chan struct{}, <-chan string) {

	var (
		done     = make(chan struct{})
		messages = make(chan string)
	)

	cleanup := func() {
		defer close(done)

		close(messages)
		log.Println("hasPapers: shutdown complete")
	}

	go func() {
		defer cleanup()

		for {
			var hasTobacco bool
			var hasLighter bool

			for !(hasTobacco && hasLighter) {
				select {
				case <-stop:
					return
				case <-tobaccoCh:
					hasTobacco = true
				case <-lighterCh:
					hasLighter = true
				}
			}

			msg := fmt.Sprintf("hasPapers: rolled and smoked a cigarette")

			select {
			case <-stop:
				return
			case messages <- msg:
			}
		}
	}()

	return done, messages
}

func hasTobacco(
	stop <-chan struct{},
	papersCh <-chan Papers,
	lighterCh <-chan Lighter,
) (<-chan struct{}, <-chan string) {

	var (
		done     = make(chan struct{})
		messages = make(chan string)
	)

	cleanup := func() {
		defer close(done)

		close(messages)
		log.Println("hasTobacco: shutdown complete")
	}

	go func() {
		defer cleanup()

		for {
			var hasPapers bool
			var hasLighter bool

			for !(hasPapers && hasLighter) {
				select {
				case <-stop:
					return
				case <-papersCh:
					hasPapers = true
				case <-lighterCh:
					hasLighter = true
				}
			}

			msg := fmt.Sprintf("hasTobacco: rolled and smoked a cigarette")

			select {
			case <-stop:
				return
			case messages <- msg:
			}
		}
	}()

	return done, messages
}

func run(stop <-chan struct{}, delay time.Duration) <-chan struct{} {
	done := make(chan struct{})

	signal := make(chan struct{})

	agentDone, forTable := agent(stop, signal)

	tableDone, forSWT, forSWP, forSWL := table(stop, forTable)

	tobaccoDone, tobaccoMsgs := hasTobacco(stop, forSWT.Papers, forSWT.Lighter)

	papersDone, papersMsgs := hasPapers(stop, forSWP.Tobacco, forSWP.Lighter)

	lighterDone, lighterMsgs := hasLighter(stop, forSWL.Tobacco, forSWL.Papers)

	wait := func() {
		<-agentDone
		<-tableDone
		<-tobaccoDone
		<-papersDone
		<-lighterDone
	}

	go func() {
		defer close(done)
		defer close(signal)
		defer wait()
		defer log.Println("run: stopped reading msgs")

		signal <- struct{}{}

		for {
			select {
			case <-stop:
				return
			case msg, ok := <-tobaccoMsgs:
				if !ok {
					return
				}
				log.Println(msg)
			case msg, ok := <-papersMsgs:
				if !ok {
					return
				}
				log.Println(msg)
			case msg, ok := <-lighterMsgs:
				if !ok {
					return
				}
				log.Println(msg)
			}

			time.Sleep(delay)

			select {
			case <-stop:
				return
			case signal <- struct{}{}:
			}
		}
	}()

	return done
}

func main() {
	var (
		stop        = make(chan struct{})
		timeout     = 3 * time.Second
		delay       = timeout / 6
		isSignaling = run(stop, delay)
	)

	time.AfterFunc(timeout, func() {
		log.Println("main: shutting down")
		close(stop)
	})

	defer log.Println("main: shutdown complete")
	<-isSignaling
}
