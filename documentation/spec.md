# NTSM File Format Specification

## Overview

NTSM (Netisu Scene Mesh) a binary file format designed for storing 3D models with optional particle effects and lua/luau sctipting abilties. It's built on top of the glTF binary format (.glb) with additional sections for particle systems, scripting and textures.

## Goals

- Maintain compatibility with existing glTF rendering pipelines
- Add support for particle effects without requiring new rendering engines
- Allow for future extensions (animations, physics, and more)
- Support both CPU and GPU particle rendering

## File Structure

┌─────────────────────────────────┐
│ NTSM Header │ (192 bytes fixed)
├─────────────────────────────────┤
│ Magic: "NTSM" (4 bytes) │
│ Version: uint32 (4 bytes) │
│ Name: char[128] │
│ Flags: uint8 (bitfield) │
│ Reserved: [3] bytes │
│ GLB Offset: uint32 │
│ GLB Size: uint32 │
│ Particle Offset: uint32 │
│ Particle Size: uint32 │
│ Texture Count: uint32 │
│ Texture Table Offset: uint32 │
└─────────────────────────────────┘
│ [Padding to alignment] │
├─────────────────────────────────┤
│ Embedded glTF Binary (.glb) │ (variable, at glbOffset)
├─────────────────────────────────┤
│ Particle System Data │ (variable, at particleOffset)
├─────────────────────────────────┤
│ Embedded Particle Textures │ (optional, referenced by table)
└─────────────────────────────────┘

## Header Details (192 bytes total)

| Offset | Size | Type | Description |
|--------|------|------|-------------|
| 0      | 4    | char | Magic string: "NTSM" |
| 4      | 4    | uint32 | Format version (1) |
| 8      | 128  | char | Item name (null-padded) |
| 136    | 1    | uint8 | Flags (bitfield) |
| 137    | 3    | uint8 | Reserved (padding) |
| 140    | 4    | uint32 | Offset to GLB data |
| 144    | 4    | uint32 | Size of GLB data |
| 148    | 4    | uint32 | Offset to particle data |
| 152    | 4    | uint32 | Size of particle data |
| 156    | 4    | uint32 | Number of embedded textures |
| 160    | 4    | uint32 | Offset to texture table |

### Flags Bitfield (uint8)
| Bit | Flag | Description |
|-----|------|-------------|
| 0   | has_particles | Set if particle data is present |
| 1   | use_world_space | 0 = local space emission, 1 = world space |
| 2   | animate_uv | 0 = static UV, 1 = animate UV scroll |
| 3   | enable_collision | 0 = no collision, 1 = enable collision |
| 4-7 | reserved | Must be 0 |

## GLB Section
This section contains standard glTF binary data (.glb). It's identical to the standard glTF binary format.

- Starts at `GLBOffset`
- Size is `GLBSize`
- Contains:
  - glTF header (12 bytes)
  - JSON chunk (variable size)
  - Binary chunk (variable size)

## Particle System Data

Each particle emitter is 128 bytes:
┌─────────────────────────────────┐
│ Particle Emitter (128b) │
├─────────────────────────────────┤
│ Position: [3]float32 │
│ Direction: [3]float32 │
│ SpreadAngle: float32 │
│ EmissionRate: float32 │
│ ParticleLifetime: float32 │
│ StartSize: float32 │
│ EndSize: float32 │
│ StartColor: [4]float32 │
│ EndColor: [4]float32 │
│ VelocityMin: [3]float32 │
│ VelocityMax: [3]float32 │
│ Gravity: float32 │
│ TextureIndex: int32 │
│ BlendMode: uint8 │
│ Loop: uint8 │
│ Padding: [2]uint8 │
└─────────────────────────────────┘

### Field Details

| Field | Type | Description |
|-------|------|-------------|
| Position | [3]float32 | Local position offset from object origin |
| Direction | [3]float32 | Emission direction (normalized) |
| SpreadAngle | float32 | Maximum spread angle (radians) |
| EmissionRate | float32 | Particles per second |
| ParticleLifetime | float32 | Maximum lifetime of a particle (seconds) |
| StartSize | float32 | Starting size of particles |
| EndSize | float32 | Ending size of particles |
| StartColor | [4]float32 | Starting color (RGBA 0-1) |
| EndColor | [4]float32 | Ending color (RGBA 0-1) |
| VelocityMin | [3]float32 | Minimum initial velocity |
| VelocityMax | [3]float32 | Maximum initial velocity |
| Gravity | float32 | Gravity acceleration (Y-axis) |
| TextureIndex | int32 | Index into texture table (-1 = default spark) |
| BlendMode | uint8 | 0 = additive, 1 = alpha |
| Loop | uint8 | 0 = once, 1 = loop |

## Texture Table

The texture table maps texture indices to embedded texture data:
┌─────────────────────────────────┐
│ Texture Table Entry │
├─────────────────────────────────┤
│ Texture Name: char[64] │
│ Texture Size: uint32 │
│ Texture Offset: uint32 │
└─────────────────────────────────┘

Each texture is stored after the texture table:
┌─────────────────────────────────┐
│ Texture Data │
└─────────────────────────────────┘

## File Creation Process

1. Convert existing .obj to .glb (only of its a .obj already)
2. Build NTSM header with appropriate offsets
3. Write header (192 bytes)
4. Write padding to ensure header is exactly 192 bytes
5. Write glTF binary data
6. Write particle data (optional)
7. Write texture table and textures (optional)

## Example Workflow
Migrating "sword.obj" with a custom sparkle particle emitter config "sparkles.json" to the ntsm format.

Convert sword.obj to sword.glb
Parse sparkles.json into particle emitters
Build NTSM header
Write to sword.ntsm

## Binary Layout Example

Let's look at a simple example with one emitter:

Offset 0-3: "NTSM"
Offset 4-7: 1 (version)
Offset 8-135: "sword" (null-padded)
Offset 136: 0x01 (has particles)
Offset 137-139: 0x00 (padding)
Offset 140-143: 192 (GLB offset)
Offset 144-147: 1024 (GLB size)
Offset 148-151: 1216 (particle offset)
Offset 152-155: 128 (particle size)
Offset 156-159: 0 (texture count)
Offset 160-163: 0 (texture table offset)
... (192 bytes total header)
0x0000C0: [GLB data - 1024 bytes]
0x0400: [Particle emitter - 128 bytes]

## Error Handling

- If `has_particles` flag is set but `ParticleSize` is 0 → invalid file
- If `ParticleSize` is not a multiple of 128 → invalid file
- If `GLBSize` is too small for valid glTF → invalid file
- If `TextureCount` > 0 but `TextureTableOffset` is invalid → invalid file

## Versioning

- Version 1: Initial specification
- Future versions may add new sections or fields
- Version should always be 1 for this specification

## Tools

- `ntsm-migrate`: Converts .obj/.glb to .ntsm
- `ntsm-pack`: Creates .ntsm from glb + particles.json
- `ntsm-unpack`: Extracts glb and particles from .ntsm

## References

- [glTF Binary Format](https://github.com/KhronosGroup/glTF/blob/main/specification/2.0/README.md#glb-file-format)
- [glTF Specification](https://github.com/KhronosGroup/glTF/blob/main/specification/2.0/README.md)