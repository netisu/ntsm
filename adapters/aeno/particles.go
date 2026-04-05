package aeno

import (
	"math"
	"math/rand"

	"github.com/netisu/aeno"
	"github.com/netisu/ntsm"
)

type Particle struct {
	Position     [3]float32
	Velocity     [3]float32
	Size         float32
	StartSize    float32
	EndSize      float32
	Color        [4]float32
	StartColor   [4]float32
	EndColor     [4]float32
	Lifetime     float32
	MaxLife      float32
	TextureIndex int32
}

type CPUParticleSystem struct {
	emitters     []ntsm.ParticleEmitter
	particles    []Particle
	textures     []aeno.Texture
	textureCache map[int]*aeno.Mesh
	meshCache    map[string]*aeno.Mesh
}

var (
	// Color3(144, 25, 255) / 255.0f
	DefaultSparkleColor = [4]float32{144.0 / 255.0, 25.0 / 255.0, 1.0, 1.0}
)

func (ps *CPUParticleSystem) SetTextures(textures []aeno.Texture) {
	ps.textures = textures
}

func NewCPUParticleSystem(emitters []ntsm.ParticleEmitter) *CPUParticleSystem {
	return &CPUParticleSystem{
		emitters:     emitters,
		particles:    make([]Particle, 0),
		textureCache: make(map[int]*aeno.Mesh),
		meshCache:    make(map[string]*aeno.Mesh),
	}
}

func (ps *CPUParticleSystem) Update(deltaTime float64, modelMatrix aeno.Matrix) {
	alive := make([]Particle, 0, len(ps.particles))
	for _, p := range ps.particles {
		p.Lifetime -= float32(deltaTime)

		if p.Lifetime > 0 {
			p.Position[0] += p.Velocity[0] * float32(deltaTime)
			p.Position[1] += p.Velocity[1] * float32(deltaTime)
			p.Position[2] += p.Velocity[2] * float32(deltaTime)

			t := 1.0 - (p.Lifetime / p.MaxLife)

			p.Color = interpolateColor(p.StartColor, p.EndColor, t)
			p.Size = lerp(p.StartSize, p.EndSize, t)

			alive = append(alive, p)
		}
	}
	ps.particles = alive

	// Add new particles based on emission rate
	for _, emitter := range ps.emitters {
		worldOrigin := modelMatrix.MulPosition(aeno.V(
			float64(emitter.Position[0]),
			float64(emitter.Position[1]),
			float64(emitter.Position[2]),
		))
		spawnRate := emitter.EmissionRate * float32(deltaTime)

		for i := 0; i < int(spawnRate); i++ {
			ps.spawnParticle(emitter, worldOrigin)
		}

		// if remainder is 0.41, there's a 41% chance to spawn 1 more
		wholeParticles := int(spawnRate)
		fractionalPart := spawnRate - float32(wholeParticles)
		if rand.Float32() < fractionalPart {
			ps.spawnParticle(emitter, worldOrigin)
		}
	}
}

func (ps *CPUParticleSystem) spawnParticle(emitter ntsm.ParticleEmitter, origin aeno.Vector) {
	if emitter.EmissionRate <= 0 {
		return
	}

	if len(ps.particles) >= 40 || emitter.EmissionRate <= 0 {
		return
	}

	angle := rand.Float32() * 2 * math.Pi
	spread := rand.Float32() * emitter.SpreadAngle
	minSpeed := emitter.VelocityMin[0]
	maxSpeed := emitter.VelocityMax[0]
	if maxSpeed == 0 {
		minSpeed = 5.0
		maxSpeed = 8.0
	}
	speed := lerp(minSpeed, maxSpeed, rand.Float32())

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
		Position:     [3]float32{float32(origin.X), float32(origin.Y), float32(origin.Z)},
		Velocity:     velocity,
		Size:         emitter.StartSize,
		StartSize:    emitter.StartSize,
		EndSize:      emitter.EndSize,
		Color:        emitter.StartColor,
		StartColor:   emitter.StartColor,
		EndColor:     emitter.EndColor,
		Lifetime:     emitter.ParticleLifetime,
		MaxLife:      emitter.ParticleLifetime,
		TextureIndex: emitter.TextureIndex,
	}

	ps.particles = append(ps.particles, particle)
}

func (ps *CPUParticleSystem) GetObjects(viewMatrix aeno.Matrix) []*aeno.Object {
	objects := make([]*aeno.Object, 0, len(ps.particles))

	for _, p := range ps.particles {
		pColor := aeno.Color{
			R: float64(p.Color[0]),
			G: float64(p.Color[1]),
			B: float64(p.Color[2]),
			A: float64(p.Color[3]),
		}

		m := aeno.Identity()

		m.X03 = float64(p.Position[0])
		m.X13 = float64(p.Position[1])
		m.X23 = float64(p.Position[2])

		m.X00 = viewMatrix.X00
		m.X01 = viewMatrix.X10
		m.X02 = viewMatrix.X20
		m.X10 = viewMatrix.X01
		m.X11 = viewMatrix.X11
		m.X12 = viewMatrix.X21
		m.X20 = viewMatrix.X02
		m.X21 = viewMatrix.X12
		m.X22 = viewMatrix.X22

		s := float64(p.Size)
		finalMatrix := m.Mul(aeno.Scale(aeno.Vector{X: s, Y: s, Z: 1.0}))

		obj := &aeno.Object{
			Mesh:           createUnitSquareMesh(pColor),
			Texture: ps.textures[p.TextureIndex],
			UseVertexColor: true,
			Matrix:         finalMatrix,
		}

		objects = append(objects, obj)
	}
	return objects
}

func createUnitSquareMesh(col aeno.Color) *aeno.Mesh {
	half := 0.5

	p0 := aeno.V(-half, -half, 0)
	p1 := aeno.V(half, -half, 0)
	p2 := aeno.V(half, half, 0)
	p3 := aeno.V(-half, half, 0)

	v0 := aeno.Vertex{Position: p0, Texture: aeno.Vector{X: 0, Y: 1}, Color: col} // Top Left
	v1 := aeno.Vertex{Position: p1, Texture: aeno.Vector{X: 1, Y: 1}, Color: col} // Top Right
	v2 := aeno.Vertex{Position: p2, Texture: aeno.Vector{X: 1, Y: 0}, Color: col} // Bottom Right
	v3 := aeno.Vertex{Position: p3, Texture: aeno.Vector{X: 0, Y: 0}, Color: col} // Bottom Left

	return aeno.NewTriangleMesh([]*aeno.Triangle{
		aeno.NewTriangle(v0, v1, v2),
		aeno.NewTriangle(v0, v2, v3),
	})
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
