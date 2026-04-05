package aeno

import (
	"math"
	"math/rand"

	"github.com/netisu/aeno"
	"github.com/netisu/ntsm"
)

type Particle struct {
	Position [3]float32
	Velocity [3]float32
	Size     float32
	Color    [4]float32
	Lifetime float32
	MaxLife  float32
}

type CPUParticleSystem struct {
	emitters     []ntsm.ParticleEmitter
	particles    []Particle
	textureCache map[int]*aeno.Mesh
}

func NewCPUParticleSystem(emitters []ntsm.ParticleEmitter) *CPUParticleSystem {
	return &CPUParticleSystem{
		emitters:     emitters,
		particles:    make([]Particle, 0),
		textureCache: make(map[int]*aeno.Mesh),
	}
}

func (ps *CPUParticleSystem) Update(deltaTime float64) {
	// Remove dead particles
	alive := make([]Particle, 0, len(ps.particles))
	for _, p := range ps.particles {
		p.Lifetime -= float32(deltaTime)
		if p.Lifetime > 0 {
			alive = append(alive, p)
		}
	}
	ps.particles = alive

	// Add new particles based on emission rate
	for _, emitter := range ps.emitters {
		spawnRate := emitter.EmissionRate * float32(deltaTime)
		for i := 0; i < int(spawnRate); i++ {
			ps.spawnParticle(emitter)
		}
	}
}

func (ps *CPUParticleSystem) spawnParticle(emitter ntsm.ParticleEmitter) {
	angle := rand.Float32() * 2 * math.Pi
	spread := rand.Float32() * emitter.SpreadAngle
	speed := lerp(emitter.VelocityMin[0], emitter.VelocityMax[0], rand.Float32())

	dx := float32(math.Cos(float64(angle))) * spread
	dy := rand.Float32() * spread
	dz := float32(math.Sin(float64(angle))) * spread

	velocity := [3]float32{
		emitter.Direction[0] + dx,
		emitter.Direction[1] + dy,
		emitter.Direction[2] + dz,
	}

	length := float32(math.Sqrt(float64(velocity[0]*velocity[0] + velocity[1]*velocity[1] + velocity[2]*velocity[2])))
	if length > 0 {
		velocity[0] *= speed / length
		velocity[1] *= speed / length
		velocity[2] *= speed / length
	}

	particle := Particle{
		Position: emitter.Position,
		Velocity: velocity,
		Size:     emitter.StartSize,
		Color:    interpolateColor(emitter.StartColor, emitter.EndColor, 0),
		Lifetime: emitter.ParticleLifetime,
		MaxLife:  emitter.ParticleLifetime,
	}

	ps.particles = append(ps.particles, particle)
}

func (ps *CPUParticleSystem) GetObjects() []*aeno.Object {
	objects := make([]*aeno.Object, 0, len(ps.particles))

	for _, p := range ps.particles {
		size := p.Size
		color := aeno.Color{R: float64(p.Color[0]), G: float64(p.Color[1]), B: float64(p.Color[2]), A: float64(p.Color[3])}

		mesh := createSquareMesh(size)
		obj := &aeno.Object{
			Mesh:   mesh,
			Color:  color,
			Matrix: aeno.Translate(aeno.V(float64(p.Position[0]), float64(p.Position[1]), float64(p.Position[2]))),
		}

		objects = append(objects, obj)
	}

	return objects
}

func createSquareMesh(size float32) *aeno.Mesh {
	half := float64(size) / 2
	vertices := []aeno.Vector{
		aeno.V(-half, -half, 0), aeno.V(half, -half, 0),
		aeno.V(half, half, 0), aeno.V(-half, half, 0),
	}

	triangles := []*aeno.Triangle{
		aeno.NewTriangleForPoints(vertices[0], vertices[1], vertices[2]),
		aeno.NewTriangleForPoints(vertices[0], vertices[2], vertices[3]),
	}

	return aeno.NewTriangleMesh(triangles)
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func interpolateColor(start, end [4]float32, t float32) [4]float32 {
	return [4]float32{
		lerp(start[0], end[0], t),
		lerp(start[1], end[1], t),
		lerp(start[2], end[2], t),
		lerp(start[3], end[3], t),
	}
}