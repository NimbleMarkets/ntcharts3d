struct Uniforms {
    view_proj: mat4x4<f32>,
    viewport: vec2<f32>,
    point_size: f32,
    _pad: f32,
};

struct Point {
    pos: vec4<f32>,
    color: vec4<f32>,
};

@group(0) @binding(0) var<uniform> u: Uniforms;
@group(0) @binding(1) var<storage, read> points: array<Point>;

struct VSOut {
    @builtin(position) clip: vec4<f32>,
    @location(0) color: vec3<f32>,
    @location(1) uv: vec2<f32>,
};

// One instance per point, six vertices per instance forming a screen-space
// quad of radius point_size pixels. The offset is scaled by clip.w so it
// survives the perspective divide unchanged.
@vertex
fn vs_main(@builtin(vertex_index) vi: u32, @builtin(instance_index) ii: u32) -> VSOut {
    var corners = array<vec2<f32>, 6>(
        vec2<f32>(-1.0, -1.0), vec2<f32>(1.0, -1.0), vec2<f32>(1.0, 1.0),
        vec2<f32>(-1.0, -1.0), vec2<f32>(1.0, 1.0), vec2<f32>(-1.0, 1.0));
    let p = points[ii];
    let corner = corners[vi];
    var clip = u.view_proj * vec4<f32>(p.pos.xyz, 1.0);
    let offset = corner * u.point_size * 2.0 / u.viewport * clip.w;
    clip = vec4<f32>(clip.xy + offset, clip.zw);
    var out: VSOut;
    out.clip = clip;
    out.color = p.color.rgb;
    out.uv = corner;
    return out;
}

@fragment
fn fs_main(in: VSOut) -> @location(0) vec4<f32> {
    if (dot(in.uv, in.uv) > 1.0) {
        discard;
    }
    return vec4<f32>(in.color, 1.0);
}
