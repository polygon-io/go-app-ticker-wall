//! Easing functions (normalized Penner equations, input/output in `[0, 1]`).
//! These replace the Go `fogleman/ease` dependency; only the variants the
//! announcement animations use are implemented.

use std::f32::consts::PI;

const C1: f32 = 1.70158;
const C3: f32 = C1 + 1.0;
const C4: f32 = (2.0 * PI) / 3.0;

pub fn in_quint(t: f32) -> f32 {
    t * t * t * t * t
}

pub fn out_quint(t: f32) -> f32 {
    1.0 - (1.0 - t).powi(5)
}

pub fn in_back(t: f32) -> f32 {
    C3 * t * t * t - C1 * t * t
}

pub fn out_back(t: f32) -> f32 {
    1.0 + C3 * (t - 1.0).powi(3) + C1 * (t - 1.0).powi(2)
}

pub fn in_elastic(t: f32) -> f32 {
    if t == 0.0 {
        0.0
    } else if t == 1.0 {
        1.0
    } else {
        -(2.0_f32.powf(10.0 * t - 10.0)) * ((t * 10.0 - 10.75) * C4).sin()
    }
}

pub fn out_elastic(t: f32) -> f32 {
    if t == 0.0 {
        0.0
    } else if t == 1.0 {
        1.0
    } else {
        2.0_f32.powf(-10.0 * t) * ((t * 10.0 - 0.75) * C4).sin() + 1.0
    }
}

pub fn out_bounce(t: f32) -> f32 {
    const N1: f32 = 7.5625;
    const D1: f32 = 2.75;
    if t < 1.0 / D1 {
        N1 * t * t
    } else if t < 2.0 / D1 {
        let t = t - 1.5 / D1;
        N1 * t * t + 0.75
    } else if t < 2.5 / D1 {
        let t = t - 2.25 / D1;
        N1 * t * t + 0.9375
    } else {
        let t = t - 2.625 / D1;
        N1 * t * t + 0.984375
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn endpoints_are_zero_and_one() {
        for f in [
            in_quint as fn(f32) -> f32,
            out_quint,
            in_back,
            out_back,
            in_elastic,
            out_elastic,
            out_bounce,
        ] {
            assert!((f(0.0)).abs() < 1e-4, "f(0) should be ~0");
            assert!((f(1.0) - 1.0).abs() < 1e-4, "f(1) should be ~1");
        }
    }
}
