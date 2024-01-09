package dsim

import (
	"encoding/json"
	"fmt"
	"os"
)

type Lefs struct {
	// Slice de transiciones de esta subred
	Network TransitionMap
	//ii_indice int32	// Contador de transiciones agnadidas, Necesario ???
	// Identificadores de las transiciones sensibilizadas para
	// T = Reloj local actual. Slice que funciona como Stack
	Sensitized TransitionStack
}

func Load(filename string) (Lefs, error) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open json lefs file: %v\n", err)
		return Lefs{}, err
	}
	defer file.Close()

	result := Lefs{}
	if err := json.NewDecoder(file).Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Decode json 		file: %v\n", err)
		return Lefs{}, err
	}

	result.Sensitized = MakeTransitionStack(100) //aun siendo dinamicos...
	// result.IaIULNodes = make(map[IndLocalTrans]IndLocalTrans)
	// result.IaPULNodes = make(map[IndLocalTrans]string)

	return result, nil
}

func (l *Lefs) UnmarshalJSON(b []byte) error {

	var m map[string]*json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		return err
	}

	var transitionList []Transition
	if err := json.Unmarshal(*m["ia_red"], &transitionList); err != nil {
		return err
	}
	l.Network = make(TransitionMap, len(transitionList))
	for i, t := range transitionList {
		l.Network[t.Id] = &transitionList[i]

	}
	return nil
}

func (l Lefs) String() string {
	return fmt.Sprintf("Lef: %+v\n", l.Network)
}

func (l *Lefs) addSensitized(transitionId TransitionId) bool {
	l.Sensitized.push(transitionId)
	return true // OK
}

func (l *Lefs) updateSensitized(clock Clock) bool {
	for i, t := range (*l).Network {
		if t.Value <= 0 && t.Clock == clock {
			(*l).Sensitized.push(i)
		}
	}
	return true
}

func (l *Lefs) getSensitized() TransitionId {
	if (*l).Sensitized.isEmpty() {
		return -1
	}
	return (*l).Sensitized.pop()
}
