struct Uniforms { matrix: mat4x4<f32>, viewport: vec4<f32>, light: vec4<f32>, reserved: vec4<f32> };
struct Item { a: vec4<f32>, b: vec4<f32>, color: vec4<f32> };
@group(0) @binding(0) var<uniform> u: Uniforms;
@group(0) @binding(1) var<storage, read> items: array<Item>;
struct Out { @builtin(position) clip: vec4<f32>, @location(0) color: vec4<f32>, @location(1) uv: vec2<f32>, @location(2) @interpolate(flat) point: f32 };
fn output(pos: vec3<f32>, normal: vec3<f32>, color: vec4<f32>) -> Out {
 var o: Out; o.clip=u.matrix*vec4<f32>(pos,1.0); var light=1.0;
 if dot(normal,normal)>0.0 { light=u.light.w+(1.0-u.light.w)*max(0.0,dot(normalize(normal),normalize(u.light.xyz))); }
 o.color=vec4<f32>(color.rgb*light,1.0);o.uv=vec2<f32>(0.0);o.point=0.0;return o;
}
@vertex fn vs_point(@builtin(vertex_index) vi:u32,@builtin(instance_index) ii:u32)->Out {
 var corners=array<vec2<f32>,6>(vec2<f32>(-1.,-1.),vec2<f32>(1.,-1.),vec2<f32>(1.,1.),vec2<f32>(-1.,-1.),vec2<f32>(1.,1.),vec2<f32>(-1.,1.));
 let p=items[ii];let corner=corners[vi];var o=output(p.a.xyz,vec3<f32>(0.0),p.color);
 o.clip=vec4<f32>(o.clip.xy+corner*p.a.w*2.0/u.viewport.xy*o.clip.w,o.clip.zw);o.uv=corner;o.point=1.0;return o;
}
@vertex fn vs_mesh(@builtin(vertex_index) vi:u32)->Out {let p=items[vi];return output(p.a.xyz,p.b.xyz,p.color);}
fn lineOutput(vi:u32,ii:u32,bias:f32)->Out {
 let a=items[ii*2u];let b=items[ii*2u+1u];
 let ca=u.matrix*vec4<f32>(a.a.xyz,1.0);let cb=u.matrix*vec4<f32>(b.a.xyz,1.0);
 let delta=(cb.xy/cb.w-ca.xy/ca.w)*u.viewport.xy;
 let length2=dot(delta,delta);
 var perpendicular=vec2<f32>(0.0,1.0);
 if length2>0.000001 { perpendicular=vec2<f32>(-delta.y,delta.x)*inverseSqrt(length2); }
 let atEnd=vi==2u || vi==4u || vi==5u;
 let side=select(-1.0,1.0,vi==1u || vi==2u || vi==4u);
 var o=output(select(a.a.xyz,b.a.xyz,atEnd),vec3<f32>(0.0),select(a.color,b.color,atEnd));
 o.clip=vec4<f32>(o.clip.xy+perpendicular*side/u.viewport.xy*o.clip.w,o.clip.z+bias*o.clip.w,o.clip.w);
 return o;
}
@vertex fn vs_line(@builtin(vertex_index) vi:u32,@builtin(instance_index) ii:u32)->Out { return lineOutput(vi,ii,-0.00001); }
@vertex fn vs_grid(@builtin(vertex_index) vi:u32,@builtin(instance_index) ii:u32)->Out { return lineOutput(vi,ii,0.00002); }
@vertex fn vs_ui(@builtin(vertex_index) vi:u32)->Out {let p=items[vi];var o:Out;let x=p.a.x/u.viewport.x*2.0-1.0;let y=1.0-p.a.y/u.viewport.y*2.0;o.clip=vec4<f32>(x,y,0.0,1.0);o.color=p.color;o.uv=vec2<f32>(0.0);o.point=0.0;return o;}
@vertex fn vs_box(@builtin(vertex_index) vi:u32,@builtin(instance_index) ii:u32)->Out {
var corners=array<vec3<f32>,36>(vec3<f32>(0.0,0.0,0.0),vec3<f32>(0.0,1.0,0.0),vec3<f32>(1.0,1.0,0.0),vec3<f32>(0.0,0.0,0.0),vec3<f32>(1.0,1.0,0.0),vec3<f32>(1.0,0.0,0.0),vec3<f32>(0.0,0.0,1.0),vec3<f32>(1.0,0.0,1.0),vec3<f32>(1.0,1.0,1.0),vec3<f32>(0.0,0.0,1.0),vec3<f32>(1.0,1.0,1.0),vec3<f32>(0.0,1.0,1.0),vec3<f32>(0.0,0.0,0.0),vec3<f32>(1.0,0.0,0.0),vec3<f32>(1.0,0.0,1.0),vec3<f32>(0.0,0.0,0.0),vec3<f32>(1.0,0.0,1.0),vec3<f32>(0.0,0.0,1.0),vec3<f32>(0.0,1.0,0.0),vec3<f32>(0.0,1.0,1.0),vec3<f32>(1.0,1.0,1.0),vec3<f32>(0.0,1.0,0.0),vec3<f32>(1.0,1.0,1.0),vec3<f32>(1.0,1.0,0.0),vec3<f32>(0.0,0.0,0.0),vec3<f32>(0.0,0.0,1.0),vec3<f32>(0.0,1.0,1.0),vec3<f32>(0.0,0.0,0.0),vec3<f32>(0.0,1.0,1.0),vec3<f32>(0.0,1.0,0.0),vec3<f32>(1.0,0.0,0.0),vec3<f32>(1.0,1.0,0.0),vec3<f32>(1.0,1.0,1.0),vec3<f32>(1.0,0.0,0.0),vec3<f32>(1.0,1.0,1.0),vec3<f32>(1.0,0.0,1.0));
var normals=array<vec3<f32>,6>(vec3<f32>(0.0,0.0,-1.0),vec3<f32>(0.0,0.0,1.0),vec3<f32>(0.0,-1.0,0.0),vec3<f32>(0.0,1.0,0.0),vec3<f32>(-1.0,0.0,0.0),vec3<f32>(1.0,0.0,0.0));
let p=items[ii];return output(p.a.xyz+corners[vi]*p.b.xyz,normals[vi/6u],p.color);}
@fragment fn fs_main(o:Out)->@location(0) vec4<f32> {if o.point>0.0 && dot(o.uv,o.uv)>1.0 {discard;} return o.color;}

@group(1) @binding(0) var mapTexture: texture_2d<f32>;
@group(1) @binding(1) var mapSampler: sampler;
@vertex fn vs_texture(@builtin(vertex_index) vi:u32)->Out {
 let p=items[vi];var o=output(p.a.xyz,p.b.xyz,p.color);o.uv=vec2<f32>(p.a.w,p.b.w);return o;
}
@fragment fn fs_texture(o:Out)->@location(0) vec4<f32> {
 return vec4<f32>(textureSample(mapTexture,mapSampler,o.uv).rgb*o.color.rgb,1.0);
}
