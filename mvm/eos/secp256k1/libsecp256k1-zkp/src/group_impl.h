/**********************************************************************
 * Copyright (c) 2013, 2014 Pieter Wuille                             *
 * Distributed under the MIT software license, see the accompanying   *
 * file COPYING or http://www.opensource.org/licenses/mit-license.php.*
 **********************************************************************/

#ifndef _SECP256K1_GROUP_IMPL_H_
#define _SECP256K1_GROUP_IMPL_H_

#include <string.h>

#include "num.h"
#include "field.h"
#include "group.h"

/** Generator for secp256k1, value 'g' defined in
 *  "Standards for Efficient Cryptography" (SEC2) 2.7.1.
 */
static const secp256k1_ge_t secp256k1_ge_const_g = SECP256K1_GE_CONST(
    0x79BE667EUL, 0xF9DCBBACUL, 0x55A06295UL, 0xCE870B07UL,
    0x029BFCDBUL, 0x2DCE28D9UL, 0x59F2815BUL, 0x16F81798UL,
    0x483ADA77UL, 0x26A3C465UL, 0x5DA4FBFCUL, 0x0E1108A8UL,
    0xFD17B448UL, 0xA6855419UL, 0x9C47D08FUL, 0xFB10D4B8UL
);

/** Alternative generator for secp256k1.
 *  This is the sha256 of 'g' after DER encoding (without compression),
 *  which happens to be a point on the curve.
 *  sage: G2 = EllipticCurve ([F (0), F (7)]).lift_x(int(hashlib.sha256('0479be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8'.decode('hex')).hexdigest(),16))
 *  sage: '%x %x'%G2.xy()
 */
static const secp256k1_ge_t secp256k1_ge_const_g2 = SECP256K1_GE_CONST(
    0x50929b74UL, 0xc1a04954UL, 0xb78b4b60UL, 0x35e97a5eUL,
    0x078a5a0fUL, 0x28ec96d5UL, 0x47bfee9aUL, 0xce803ac0UL,
    0x31d3c686UL, 0x3973926eUL, 0x049e637cUL, 0xb1b5f40aUL,
    0x36dac28aUL, 0xf1766968UL, 0xc30c2313UL, 0xf3a38904UL
);

static void secp256k1_ge_set_gej_zinv(secp256k1_ge_t *r, const secp256k1_gej_t *a, const secp256k1_fe_t *zi) {
    secp256k1_fe_t zi2; 
    secp256k1_fe_t zi3;
    secp256k1_fe_sqr(&zi2, zi);
    secp256k1_fe_mul(&zi3, &zi2, zi);
    secp256k1_fe_mul(&r->x, &a->x, &zi2);
    secp256k1_fe_mul(&r->y, &a->y, &zi3);
    r->infinity = a->infinity;
}

static void secp256k1_ge_set_infinity(secp256k1_ge_t *r) {
    r->infinity = 1;
}

static void secp256k1_ge_set_xy(secp256k1_ge_t *r, const secp256k1_fe_t *x, const secp256k1_fe_t *y) {
    r->infinity = 0;
    r->x = *x;
    r->y = *y;
}

static int secp256k1_ge_is_infinity(const secp256k1_ge_t *a) {
    return a->infinity;
}

static void secp256k1_ge_neg(secp256k1_ge_t *r, const secp256k1_ge_t *a) {
    *r = *a;
    secp256k1_fe_normalize_weak(&r->y);
    secp256k1_fe_negate(&r->y, &r->y, 1);
}

static void secp256k1_ge_set_gej(secp256k1_ge_t *r, secp256k1_gej_t *a) {
    secp256k1_fe_t z2, z3;
    r->infinity = a->infinity;
    secp256k1_fe_inv(&a->z, &a->z);
    secp256k1_fe_sqr(&z2, &a->z);
    secp256k1_fe_mul(&z3, &a->z, &z2);
    secp256k1_fe_mul(&a->x, &a->x, &z2);
    secp256k1_fe_mul(&a->y, &a->y, &z3);
    secp256k1_fe_set_int(&a->z, 1);
    r->x = a->x;
    r->y = a->y;
}

static void secp256k1_ge_set_gej_var(secp256k1_ge_t *r, secp256k1_gej_t *a) {
    secp256k1_fe_t z2, z3;
    r->infinity = a->infinity;
    if (a->infinity) {
        return;
    }
    secp256k1_fe_inv_var(&a->z, &a->z);
    secp256k1_fe_sqr(&z2, &a->z);
    secp256k1_fe_mul(&z3, &a->z, &z2);
    secp256k1_fe_mul(&a->x, &a->x, &z2);
    secp256k1_fe_mul(&a->y, &a->y, &z3);
    secp256k1_fe_set_int(&a->z, 1);
    r->x = a->x;
    r->y = a->y;
}

static void secp256k1_ge_set_all_gej_var(size_t len, secp256k1_ge_t *r, const secp256k1_gej_t *a) {
    secp256k1_fe_t *az;
    secp256k1_fe_t *azi;
    size_t i;
    size_t count = 0;
    az = (secp256k1_fe_t *)checked_malloc(sizeof(secp256k1_fe_t) * len);
    for (i = 0; i < len; i++) {
        if (!a[i].infinity) {
            az[count++] = a[i].z;
        }
    }

    azi = (secp256k1_fe_t *)checked_malloc(sizeof(secp256k1_fe_t) * count);
    secp256k1_fe_inv_all_var(count, azi, az);
    free(az);

    count = 0;
    for (i = 0; i < len; i++) {
        r[i].infinity = a[i].infinity;
        if (!a[i].infinity) {
            secp256k1_ge_set_gej_zinv(&r[i], &a[i], &azi[count++]);
        }
    }
    free(azi);
}

static void secp256k1_ge_set_table_gej_var(size_t len, secp256k1_ge_t *r, const secp256k1_gej_t *a, const secp256k1_fe_t *zr) {
    size_t i = len - 1;
    secp256k1_fe_t zi;

    if (len < 1)
        return;

    /* Compute the inverse of the last z coordinate, and use it to compute the last affine output. */
    secp256k1_fe_inv(&zi, &a[i].z);
    secp256k1_ge_set_gej_zinv(&r[i], &a[i], &zi);

    /* Work out way backwards, using the z-ratios to scale the x/y values. */
    while (i > 0) {
        secp256k1_fe_mul(&zi, &zi, &zr[i]);
        i--;
        secp256k1_ge_set_gej_zinv(&r[i], &a[i], &zi);
    }
}

static void secp256k1_ge_globalz_set_table_gej(size_t len, secp256k1_ge_t *r, secp256k1_fe_t *globalz, const secp256k1_gej_t *a, const secp256k1_fe_t *zr) {
    size_t i = len - 1;
    secp256k1_fe_t zs;

    if (len < 1)
        return;

    /* The z of the final point gives us the "global Z" for the table. */
    r[i].x = a[i].x;
    r[i].y = a[i].y;
    *globalz = a[i].z;
    r[i].infinity = 0;
    zs = zr[i];

    /* Work our way backwards, using the z-ratios to scale the x/y values. */
    while (i > 0) {
        if (i != len - 1) {
            secp256k1_fe_mul(&zs, &zs, &zr[i]);
        }
        i--;
        secp256k1_ge_set_gej_zinv(&r[i], &a[i], &zs);
    }
}

static void secp256k1_gej_set_infinity(secp256k1_gej_t *r) {
    r->infinity = 1;
    secp256k1_fe_set_int(&r->x, 0);
    secp256k1_fe_set_int(&r->y, 0);
    secp256k1_fe_set_int(&r->z, 0);
}

static void secp256k1_gej_set_xy(secp256k1_gej_t *r, const secp256k1_fe_t *x, const secp256k1_fe_t *y) {
    r->infinity = 0;
    r->x = *x;
    r->y = *y;
    secp256k1_fe_set_int(&r->z, 1);
}

static void secp256k1_gej_clear(secp256k1_gej_t *r) {
    r->infinity = 0;
    secp256k1_fe_clear(&r->x);
    secp256k1_fe_clear(&r->y);
    secp256k1_fe_clear(&r->z);
}

static void secp256k1_ge_clear(secp256k1_ge_t *r) {
    r->infinity = 0;
    secp256k1_fe_clear(&r->x);
    secp256k1_fe_clear(&r->y);
}

static int secp256k1_ge_set_xo_var(secp256k1_ge_t *r, const secp256k1_fe_t *x, int odd) {
    secp256k1_fe_t x2, x3, c;
    r->x = *x;
    secp256k1_fe_sqr(&x2, x);
    secp256k1_fe_mul(&x3, x, &x2);
    r->infinity = 0;
    secp256k1_fe_set_int(&c, 7);
    secp256k1_fe_add(&c, &x3);
    if (!secp256k1_fe_sqrt_var(&r->y, &c)) {
        return 0;
    }
    secp256k1_fe_normalize_var(&r->y);
    if (secp256k1_fe_is_odd(&r->y) != odd) {
        secp256k1_fe_negate(&r->y, &r->y, 1);
    }
    return 1;
}

static void secp256k1_gej_set_ge(secp256k1_gej_t *r, const secp256k1_ge_t *a) {
   r->infinity = a->infinity;
   r->x = a->x;
   r->y = a->y;
   secp256k1_fe_set_int(&r->z, 1);
}

static int secp256k1_gej_eq_x_var(const secp256k1_fe_t *x, const secp256k1_gej_t *a) {
    secp256k1_fe_t r, r2;
    VERIFY_CHECK(!a->infinity);
    secp256k1_fe_sqr(&r, &a->z); secp256k1_fe_mul(&r, &r, x);
    r2 = a->x; secp256k1_fe_normalize_weak(&r2);
    return secp256k1_fe_equal_var(&r, &r2);
}

static void secp256k1_gej_neg(secp256k1_gej_t *r, const secp256k1_gej_t *a) {
    r->infinity = a->infinity;
    r->x = a->x;
    r->y = a->y;
    r->z = a->z;
    secp256k1_fe_normalize_weak(&r->y);
    secp256k1_fe_negate(&r->y, &r->y, 1);
}

static int secp256k1_gej_is_infinity(const secp256k1_gej_t *a) {
    return a->infinity;
}

static int secp256k1_gej_is_valid_var(const secp256k1_gej_t *a) {
    secp256k1_fe_t y2, x3, z2, z6;
    if (a->infinity) {
        return 0;
    }
    /** y^2 = x^3 + 7
     *  (Y/Z^3)^2 = (X/Z^2)^3 + 7
     *  Y^2 / Z^6 = X^3 / Z^6 + 7
     *  Y^2 = X^3 + 7*Z^6
     */
    secp256k1_fe_sqr(&y2, &a->y);
    secp256k1_fe_sqr(&x3, &a->x); secp256k1_fe_mul(&x3, &x3, &a->x);
    secp256k1_fe_sqr(&z2, &a->z);
    secp256k1_fe_sqr(&z6, &z2); secp256k1_fe_mul(&z6, &z6, &z2);
    secp256k1_fe_mul_int(&z6, 7);
    secp256k1_fe_add(&x3, &z6);
    secp256k1_fe_normalize_weak(&x3);
    return secp256k1_fe_equal_var(&y2, &x3);
}

static int secp256k1_ge_is_valid_var(const secp256k1_ge_t *a) {
    secp256k1_fe_t y2, x3, c;
    if (a->infinity) {
        return 0;
    }
    /* y^2 = x^3 + 7 */
    secp256k1_fe_sqr(&y2, &a->y);
    secp256k1_fe_sqr(&x3, &a->x); secp256k1_fe_mul(&x3, &x3, &a->x);
    secp256k1_fe_set_int(&c, 7);
    secp256k1_fe_add(&x3, &c);
    secp256k1_fe_normalize_weak(&x3);
    return secp256k1_fe_equal_var(&y2, &x3);
}

static void secp256k1_gej_double_var(secp256k1_gej_t *r, const secp256k1_gej_t *a, secp256k1_fe_t *rzr) {
    /* Operations: 3 mul, 4 sqr, 0 normalize, 12 mul_int/add/negate */
    secp256k1_fe_t t1,t2,t3,t4;
    /** For secp256k1, 2Q is infinity if and only if Q is infinity. This is because if 2Q = infinity,
     *  Q must equal -Q, or that Q.y == -(Q.y), or Q.y is 0. For a point on y^2 = x^3 + 7 to have
     *  y=0, x^3 must be -7 mod p. However, -7 has no cube root mod p.
     */
    r->infinity = a->infinity;
    if (r->infinity) {
        if (rzr) {
            secp256k1_fe_set_int(rzr, 1);
        }
        return;
    }

    if (rzr) {
        *rzr = a->y;
        secp256k1_fe_normalize_weak(rzr);
        secp256k1_fe_mul_int(rzr, 2);
    }

    secp256k1_fe_mul(&r->z, &a->z, &a->y);
    secp256k1_fe_mul_int(&r->z, 2);       /* Z' = 2*Y*Z (2) */
    secp256k1_fe_sqr(&t1, &a->x);
    secp256k1_fe_mul_int(&t1, 3);         /* T1 = 3*X^2 (3) */
    secp256k1_fe_sqr(&t2, &t1);           /* T2 = 9*X^4 (1) */
    secp256k1_fe_sqr(&t3, &a->y);
    secp256k1_fe_mul_int(&t3, 2);         /* T3 = 2*Y^2 (2) */
    secp256k1_fe_sqr(&t4, &t3);
    secp256k1_fe_mul_int(&t4, 2);         /* T4 = 8*Y^4 (2) */
    secp256k1_fe_mul(&t3, &t3, &a->x);    /* T3 = 2*X*Y^2 (1) */
    r->x = t3;
    secp256k1_fe_mul_int(&r->x, 4);       /* X' = 8*X*Y^2 (4) */
    secp256k1_fe_negate(&r->x, &r->x, 4); /* X' = -8*X*Y^2 (5) */
    secp256k1_fe_add(&r->x, &t2);         /* X' = 9*X^4 - 8*X*Y^2 (6) */
    secp256k1_fe_negate(&t2, &t2, 1);     /* T2 = -9*X^4 (2) */
    secp256k1_fe_mul_int(&t3, 6);         /* T3 = 12*X*Y^2 (6) */
    secp256k1_fe_add(&t3, &t2);           /* T3 = 12*X*Y^2 - 9*X^4 (8) */
    secp256k1_fe_mul(&r->y, &t1, &t3);    /* Y' = 36*X^3*Y^2 - 27*X^6 (1) */
    secp256k1_fe_negate(&t2, &t4, 2);     /* T2 = -8*Y^4 (3) */
    secp256k1_fe_add(&r->y, &t2);         /* Y' = 36*X^3*Y^2 - 27*X^6 - 8*Y^4 (4) */
}

static void secp256k1_gej_add_var(secp256k1_gej_t *r, const secp256k1_gej_t *a, const secp256k1_gej_t *b, secp256k1_fe_t *rzr) {
    /* Operations: 12 mul, 4 sqr, 2 normalize, 12 mul_int/add/negate */
    secp256k1_fe_t z22, z12, u1, u2, s1, s2, h, i, i2, h2, h3, t;

    if (a->infinity) {
        VERIFY_CHECK(rzr == NULL);
        *r = *b;
        return;
    }

    if (b->infinity) {
        if (rzr) {
            secp256k1_fe_set_int(rzr, 1);
        }
        *r = *a;
        return;
    }

    r->infinity = 0;
    secp256k1_fe_sqr(&z22, &b->z);
    secp256k1_fe_sqr(&z12, &a->z);
    secp256k1_fe_mul(&u1, &a->x, &z22);
    secp256k1_fe_mul(&u2, &b->x, &z12);
    secp256k1_fe_mul(&s1, &a->y, &z22); secp256k1_fe_mul(&s1, &s1, &b->z);
    secp256k1_fe_mul(&s2, &b->y, &z12); secp256k1_fe_mul(&s2, &s2, &a->z);
    secp256k1_fe_negate(&h, &u1, 1); secp256k1_fe_add(&h, &u2);
    secp256k1_fe_negate(&i, &s1, 1); secp256k1_fe_add(&i, &s2);
    if (secp256k1_fe_normalizes_to_zero_var(&h)) {
        if (secp256k1_fe_normalizes_to_zero_var(&i)) {
            secp256k1_gej_double_var(r, a, rzr);
        } else {
            if (rzr) {
                secp256k1_fe_set_int(rzr, 0);
            }
            r->infinity = 1;
        }
        return;
    }
    secp256k1_fe_sqr(&i2, &i);
    secp256k1_fe_sqr(&h2, &h);
    secp256k1_fe_mul(&h3, &h, &h2);
    secp256k1_fe_mul(&h, &h, &b->z);
    if (rzr) {
        *rzr = h;
    }
    secp256k1_fe_mul(&r->z, &a->z, &h);
    secp256k1_fe_mul(&t, &u1, &h2);
    r->x = t; secp256k1_fe_mul_int(&r->x, 2); secp256k1_fe_add(&r->x, &h3); secp256k1_fe_negate(&r->x, &r->x, 3); secp256k1_fe_add(&r->x, &i2);
    secp256k1_fe_negate(&r->y, &r->x, 5); secp256k1_fe_add(&r->y, &t); secp256k1_fe_mul(&r->y, &r->y, &i);
    secp256k1_fe_mul(&h3, &h3, &s1); secp256k1_fe_negate(&h3, &h3, 1);
    secp256k1_fe_add(&r->y, &h3);
}

static void secp256k1_gej_add_ge_var(secp256k1_gej_t *r, const secp256k1_gej_t *a, const secp256k1_ge_t *b, secp256k1_fe_t *rzr) {
    /* 8 mul, 3 sqr, 4 normalize, 12 mul_int/add/negate */
    secp256k1_fe_t z12, u1, u2, s1, s2, h, i, i2, h2, h3, t;
    if (a->infinity) {
        VERIFY_CHECK(rzr == NULL);
        secp256k1_gej_set_ge(r, b);
        return;
    }
    if (b->infinity) {
        if (rzr) {
            secp256k1_fe_set_int(rzr, 1);
        }
        *r = *a;
        return;
    }
    r->infinity = 0;

    secp256k1_fe_sqr(&z12, &a->z);
    u1 = a->x; secp256k1_fe_normalize_weak(&u1);
    secp256k1_fe_mul(&u2, &b->x, &z12);
    s1 = a->y; secp256k1_fe_normalize_weak(&s1);
    secp256k1_fe_mul(&s2, &b->y, &z12); secp256k1_fe_mul(&s2, &s2, &a->z);
    secp256k1_fe_negate(&h, &u1, 1); secp256k1_fe_add(&h, &u2);
    secp256k1_fe_negate(&i, &s1, 1); secp256k1_fe_add(&i, &s2);
    if (secp256k1_fe_normalizes_to_zero_var(&h)) {
        if (secp256k1_fe_normalizes_to_zero_var(&i)) {
            secp256k1_gej_double_var(r, a, rzr);
        } else {
            if (rzr) {
                secp256k1_fe_set_int(rzr, 0);
            }
            r->infinity = 1;
        }
        return;
    }
    secp256k1_fe_sqr(&i2, &i);
    secp256k1_fe_sqr(&h2, &h);
    secp256k1_fe_mul(&h3, &h, &h2);
    if (rzr) {
        *rzr = h;
    }
    secp256k1_fe_mul(&r->z, &a->z, &h);
    secp256k1_fe_mul(&t, &u1, &h2);
    r->x = t; secp256k1_fe_mul_int(&r->x, 2); secp256k1_fe_add(&r->x, &h3); secp256k1_fe_negate(&r->x, &r->x, 3); secp256k1_fe_add(&r->x, &i2);
    secp256k1_fe_negate(&r->y, &r->x, 5); secp256k1_fe_add(&r->y, &t); secp256k1_fe_mul(&r->y, &r->y, &i);
    secp256k1_fe_mul(&h3, &h3, &s1); secp256k1_fe_negate(&h3, &h3, 1);
    secp256k1_fe_add(&r->y, &h3);
}

static void secp256k1_gej_add_zinv_var(secp256k1_gej_t *r, const secp256k1_gej_t *a, const secp256k1_ge_t *b, const secp256k1_fe_t *bzinv) {
    /* 9 mul, 3 sqr, 4 normalize, 12 mul_int/add/negate */
    secp256k1_fe_t az, z12, u1, u2, s1, s2, h, i, i2, h2, h3, t;

    if (b->infinity) {
        *r = *a;
        return;
    }
    if (a->infinity) {
        secp256k1_fe_t bzinv2, bzinv3;
        r->infinity = b->infinity;
        secp256k1_fe_sqr(&bzinv2, bzinv);
        secp256k1_fe_mul(&bzinv3, &bzinv2, bzinv);
        secp256k1_fe_mul(&r->x, &b->x, &bzinv2);
        secp256k1_fe_mul(&r->y, &b->y, &bzinv3);
        secp256k1_fe_set_int(&r->z, 1);
        return;
    }
    r->infinity = 0;

    /** We need to calculate (rx,ry,rz) = (ax,ay,az) + (bx,by,1/bzinv). Due to
     *  secp256k1's isomorphism we can multiply the Z coordinates on both sides
     *  by bzinv, and get: (rx,ry,rz*bzinv) = (ax,ay,az*bzinv) + (bx,by,1).
     *  This means that (rx,ry,rz) can be calculated as
     *  (ax,ay,az*bzinv) + (bx,by,1), when not applying the bzinv factor to rz.
     *  The variable az below holds the modified Z coordinate for a, which is used
     *  for the computation of rx and ry, but not for rz.
     */
    secp256k1_fe_mul(&az, &a->z, bzinv);

    secp256k1_fe_sqr(&z12, &az);
    u1 = a->x; secp256k1_fe_normalize_weak(&u1);
    secp256k1_fe_mul(&u2, &b->x, &z12);
    s1 = a->y; secp256k1_fe_normalize_weak(&s1);
    secp256k1_fe_mul(&s2, &b->y, &z12); secp256k1_fe_mul(&s2, &s2, &az);
    secp256k1_fe_negate(&h, &u1, 1); secp256k1_fe_add(&h, &u2);
    secp256k1_fe_negate(&i, &s1, 1); secp256k1_fe_add(&i, &s2);
    if (secp256k1_fe_normalizes_to_zero_var(&h)) {
        if (secp256k1_fe_normalizes_to_zero_var(&i)) {
            secp256k1_gej_double_var(r, a, NULL);
        } else {
            r->infinity = 1;
        }
        return;
    }
    secp256k1_fe_sqr(&i2, &i);
    secp256k1_fe_sqr(&h2, &h);
    secp256k1_fe_mul(&h3, &h, &h2);
    r->z = a->z; secp256k1_fe_mul(&r->z, &r->z, &h);
    secp256k1_fe_mul(&t, &u1, &h2);
    r->x = t; secp256k1_fe_mul_int(&r->x, 2); secp256k1_fe_add(&r->x, &h3); secp256k1_fe_negate(&r->x, &r->x, 3); secp256k1_fe_add(&r->x, &i2);
    secp256k1_fe_negate(&r->y, &r->x, 5); secp256k1_fe_add(&r->y, &t); secp256k1_fe_mul(&r->y, &r->y, &i);
    secp256k1_fe_mul(&h3, &h3, &s1); secp256k1_fe_negate(&h3, &h3, 1);
    secp256k1_fe_add(&r->y, &h3);
}


static void secp256k1_gej_add_ge(secp256k1_gej_t *r, const secp256k1_gej_t *a, const secp256k1_ge_t *b) {
    /* Operations: 7 mul, 5 sqr, 5 normalize, 17 mul_int/add/negate/cmov */
    static const secp256k1_fe_t fe_1 = SECP256K1_FE_CONST(0, 0, 0, 0, 0, 0, 0, 1);
    secp256k1_fe_t zz, u1, u2, s1, s2, z, t, m, n, q, rr;
    int infinity;
    VERIFY_CHECK(!b->infinity);
    VERIFY_CHECK(a->infinity == 0 || a->infinity == 1);

    /** In:
     *    Eric Brier and Marc Joye, Weierstrass Elliptic Curves and Side-Channel Attacks.
     *    In D. Naccache and P. Paillier, Eds., Public Key Cryptography, vol. 2274 of Lecture Notes in Computer Science, pages 335-345. Springer-Verlag, 2002.
     *  we find as solution for a unified addition/doubling formula:
     *    lambda = ((x1 + x2)^2 - x1 * x2 + a) / (y1 + y2), with a = 0 for secp256k1's curve equation.
     *    x3 = lambda^2 - (x1 + x2)
     *    2*y3 = lambda * (x1 + x2 - 2 * x3) - (y1 + y2).
     *
     *  Substituting x_i = Xi / Zi^2 and yi = Yi / Zi^3, for i=1,2,3, gives:
     *    U1 = X1*Z2^2, U2 = X2*Z1^2
     *    S1 = Y1*Z2^3, S2 = Y2*Z1^3
     *    Z = Z1*Z2
     *    T = U1+U2
     *    M = S1+S2
     *    Q = T*M^2
     *    R = T^2-U1*U2
     *    X3 = 4*(R^2-Q)
     *    Y3 = 4*(R*(3*Q-2*R^2)-M^4)
     *    Z3 = 2*M*Z
     *  (Note that the paper uses xi = Xi / Zi and yi = Yi / Zi instead.)
     */

    secp256k1_fe_sqr(&zz, &a->z);                       /* z = Z1^2 */
    u1 = a->x; secp256k1_fe_normalize_weak(&u1);        /* u1 = U1 = X1*Z2^2 (1) */
    secp256k1_fe_mul(&u2, &b->x, &zz);                  /* u2 = U2 = X2*Z1^2 (1) */
    s1 = a->y; secp256k1_fe_normalize_weak(&s1);        /* s1 = S1 = Y1*Z2^3 (1) */
    secp256k1_fe_mul(&s2, &b->y, &zz);                  /* s2 = Y2*Z2^2 (1) */
    secp256k1_fe_mul(&s2, &s2, &a->z);                  /* s2 = S2 = Y2*Z1^3 (1) */
    z = a->z;                                           /* z = Z = Z1*Z2 (8) */
    t = u1; secp256k1_fe_add(&t, &u2);                  /* t = T = U1+U2 (2) */
    m = s1; secp256k1_fe_add(&m, &s2);                  /* m = M = S1+S2 (2) */
    secp256k1_fe_sqr(&n, &m);                           /* n = M^2 (1) */
    secp256k1_fe_mul(&q, &n, &t);                       /* q = Q = T*M^2 (1) */
    secp256k1_fe_sqr(&n, &n);                           /* n = M^4 (1) */
    secp256k1_fe_sqr(&rr, &t);                          /* rr = T^2 (1) */
    secp256k1_fe_mul(&t, &u1, &u2); secp256k1_fe_negate(&t, &t, 1); /* t = -U1*U2 (2) */
    secp256k1_fe_add(&rr, &t);                                      /* rr = R = T^2-U1*U2 (3) */
    secp256k1_fe_sqr(&t, &rr);                                      /* t = R^2 (1) */
    secp256k1_fe_mul(&r->z, &m, &z);                                /* r->z = M*Z (1) */
    infinity = secp256k1_fe_normalizes_to_zero(&r->z) * (1 - a->infinity);
    secp256k1_fe_mul_int(&r->z, 2 * (1 - a->infinity)); /* r->z = Z3 = 2*M*Z (2) */
    r->x = t;                                           /* r->x = R^2 (1) */
    secp256k1_fe_negate(&q, &q, 1);                     /* q = -Q (2) */
    secp256k1_fe_add(&r->x, &q);                        /* r->x = R^2-Q (3) */
    secp256k1_fe_normalize(&r->x);
    secp256k1_fe_mul_int(&q, 3);                        /* q = -3*Q (6) */
    secp256k1_fe_mul_int(&t, 2);                        /* t = 2*R^2 (2) */
    secp256k1_fe_add(&t, &q);                           /* t = 2*R^2-3*Q (8) */
    secp256k1_fe_mul(&t, &t, &rr);                      /* t = R*(2*R^2-3*Q) (1) */
    secp256k1_fe_add(&t, &n);                           /* t = R*(2*R^2-3*Q)+M^4 (2) */
    secp256k1_fe_negate(&r->y, &t, 2);                  /* r->y = R*(3*Q-2*R^2)-M^4 (3) */
    secp256k1_fe_normalize_weak(&r->y);
    secp256k1_fe_mul_int(&r->x, 4 * (1 - a->infinity)); /* r->x = X3 = 4*(R^2-Q) */
    secp256k1_fe_mul_int(&r->y, 4 * (1 - a->infinity)); /* r->y = Y3 = 4*R*(3*Q-2*R^2)-4*M^4 (4) */

    /** In case a->infinity == 1, the above code results in r->x, r->y, and r->z all equal to 0.
     *  Replace r with b->x, b->y, 1 in that case.
     */
    secp256k1_fe_cmov(&r->x, &b->x, a->infinity);
    secp256k1_fe_cmov(&r->y, &b->y, a->infinity);
    secp256k1_fe_cmov(&r->z, &fe_1, a->infinity);
    r->infinity = infinity;
}

static void secp256k1_gej_rescale(secp256k1_gej_t *r, const secp256k1_fe_t *s) {
    /* Operations: 4 mul, 1 sqr */
    secp256k1_fe_t zz;
    VERIFY_CHECK(!secp256k1_fe_is_zero(s));
    secp256k1_fe_sqr(&zz, s);
    secp256k1_fe_mul(&r->x, &r->x, &zz);                /* r->x *= s^2 */
    secp256k1_fe_mul(&r->y, &r->y, &zz);
    secp256k1_fe_mul(&r->y, &r->y, s);                  /* r->y *= s^3 */
    secp256k1_fe_mul(&r->z, &r->z, s);                  /* r->z *= s   */
}

static void secp256k1_ge_to_storage(secp256k1_ge_storage_t *r, const secp256k1_ge_t *a) {
    secp256k1_fe_t x, y;
    VERIFY_CHECK(!a->infinity);
    x = a->x;
    secp256k1_fe_normalize(&x);
    y = a->y;
    secp256k1_fe_normalize(&y);
    secp256k1_fe_to_storage(&r->x, &x);
    secp256k1_fe_to_storage(&r->y, &y);
}

static void secp256k1_ge_from_storage(secp256k1_ge_t *r, const secp256k1_ge_storage_t *a) {
    secp256k1_fe_from_storage(&r->x, &a->x);
    secp256k1_fe_from_storage(&r->y, &a->y);
    r->infinity = 0;
}

static SECP256K1_INLINE void secp256k1_ge_storage_cmov(secp256k1_ge_storage_t *r, const secp256k1_ge_storage_t *a, int flag) {
    secp256k1_fe_storage_cmov(&r->x, &a->x, flag);
    secp256k1_fe_storage_cmov(&r->y, &a->y, flag);
}

#ifdef USE_ENDOMORPHISM
static void secp256k1_ge_mul_lambda(secp256k1_ge_t *r, const secp256k1_ge_t *a) {
    static const secp256k1_fe_t beta = SECP256K1_FE_CONST(
        0x7ae96a2bul, 0x657c0710ul, 0x6e64479eul, 0xac3434e9ul,
        0x9cf04975ul, 0x12f58995ul, 0xc1396c28ul, 0x719501eeul
    );
    *r = *a;
    secp256k1_fe_mul(&r->x, &r->x, &beta);
}
#endif

#endif
