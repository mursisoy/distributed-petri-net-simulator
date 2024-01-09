package dsim

import (
	"encoding/json"
	"fmt"
	"strings"
)

type TransitionId int

type Const int

type TransitionConstant struct {
	TransitionId TransitionId
	Constant     Const
}

type Transition struct {
	// IiIdLocal en la tabla global de transiciones
	Id TransitionId `json:"ii_idglobal"`

	Value Const `json:"ii_valor"`
	Clock Clock `json:"ii_tiempo"`

	// tiempo que dura el disparo de la transicion
	Duration Clock `json:"ii_duracion_disparo"`

	// Pairwise transition-constant list inmediate update
	Update []TransitionConstant `json:"ii_listactes_IUL"`

	// Pairwise transition-constant propagation
	Propagate []TransitionConstant `json:"ii_listactes_PUL"`

	Lookahead []TransitionId

	External bool `json:"ib_desalida"`
}

func (tc *TransitionConstant) UnmarshalJSON(buf []byte) error {
	tmp := []interface{}{&tc.TransitionId, &tc.Constant}
	if err := json.Unmarshal(buf, &tmp); err != nil {
		return err
	}
	return nil
}

// actualizaTiempo modifica el tiempo de la transicion dada
func (t *Transition) updateClock(c Clock) {
	// Modificacion del tiempo
	t.Clock = c
}

func (t *Transition) updateFuncValue(value Const) {
	// Modificacion del valor de la funcion lef
	t.Value += value
}

func (t *Transition) String() string {
	return fmt.Sprintf(
		"Transition:\n"+
			"\tId: %d\n"+
			"\tValue: %d\n"+
			"\tClock: %v\n"+
			"\tDuration: %v\n"+
			"\tUpdate: %v\n"+
			"\tPropagate: %v\n",
		t.Id, t.Value, t.Clock, t.Duration,
		t.Update,
		t.Propagate)
}

// ImprimeValores de la transición
func (t *Transition) PrintValues() {
	fmt.Println("Transition -> ")
	fmt.Println("\tId: ", t.Id)
	fmt.Println("\tValue: ", t.Value)
	fmt.Println("\tClock: ", t.Clock)
}

// TransitionList is a list of transitions themselves
type TransitionMap map[TransitionId]*Transition

type TransitionStack []TransitionId

// MakeTransitionStack crea lista de tamaño aiLongitud
func MakeTransitionStack(capacidad int) TransitionStack {
	// cero length and capacidad capacity
	return make(TransitionStack, 0, capacidad)
}

// push transition id to stack
func (st *TransitionStack) push(id TransitionId) {
	*st = append(*st, id)
}

// pop transition id from stack
func (st *TransitionStack) pop() TransitionId {
	if (*st).isEmpty() {
		return -1
	}

	id := (*st)[len(*st)-1]  // obtener dato de lo alto de la pila
	*st = (*st)[:len(*st)-1] //desempilar

	return id
}

// isEmpty  the transition stack ?
func (st TransitionStack) isEmpty() bool {
	return len(st) == 0
}

func (st TransitionStack) String() string {
	if st.isEmpty() {
		return fmt.Sprintln("\tStack TRANSICIONES VACIA")
	} else {
		ret := make([]string, len(st))
		for i, tId := range st {
			ret[i] = fmt.Sprint(tId)
		}
		return strings.Join(ret, ",")
	}
}
