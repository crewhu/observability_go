package timer

import (
"testing"
"time"
)

func TestTimeFunc(t *testing.T) {
	// Criando uma função simples que dorme por um tempo específico
	sleepDuration := 50 * time.Millisecond
	fn := func() string {
		time.Sleep(sleepDuration)
		return "resultado"
	}

	// Medindo o tempo da execução
	resultado, duracao := TimeFunc(fn)

	// Verificando o resultado
	if resultado != "resultado" {
		t.Errorf("resultado esperado %s, obtido %s", "resultado", resultado)
	}

	// Verificando se a duração é pelo menos o tempo que dormimos
	// Adicionamos uma margem de erro pequena
	if duracao < sleepDuration {
		t.Errorf("duracao menor que o esperado, esperado >= %v, obtido %v", sleepDuration, duracao)
	}
}

func TestWithTiming(t *testing.T) {
	// Configurando
	sleepDuration := 50 * time.Millisecond
	var medido time.Duration
	
	fn := func() int {
		time.Sleep(sleepDuration)
		return 42
	}
	
	callback := func(d time.Duration) {
		medido = d
	}
	
	// Criando a função com medição
	timedFn := WithTiming(fn, callback)
	
	// Executando e verificando resultado
	resultado := timedFn()
	if resultado != 42 {
		t.Errorf("resultado esperado %d, obtido %d", 42, resultado)
	}
	
	// Verificando se a duração foi registrada corretamente
	if medido < sleepDuration {
		t.Errorf("duracao menor que o esperado, esperado >= %v, obtido %v", sleepDuration, medido)
	}
}

func TestMeasureTime(t *testing.T) {
	// Como MeasureTime imprime na saída padrão, não podemos testar facilmente a saída
	// Mas podemos testar se a função não causa pânico
	done := make(chan bool)
	go func() {
		defer MeasureTime("test operation")()
		time.Sleep(10 * time.Millisecond)
		done <- true
	}()
	
	select {
	case <-done:
		// Tudo certo, função executou sem pânico
	case <-time.After(100 * time.Millisecond):
		t.Error("timeout, a função MeasureTime pode ter travado")
	}
}

// Benchmark para medir o overhead da instrumentação
func BenchmarkTimeFunc(b *testing.B) {
	fn := func() int { return 42 }
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = TimeFunc(fn)
	}
}

func BenchmarkWithTiming(b *testing.B) {
	fn := func() int { return 42 }
	callback := func(d time.Duration) {}
	timedFn := WithTiming(fn, callback)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = timedFn()
	}
}
