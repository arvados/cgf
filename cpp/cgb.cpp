#include <stdio.h>
#include <stdlib.h>
#include <ctype.h>
#include <unistd.h>
#include <string.h>

#include <stdint.h>
#include <inttypes.h>

#include "cgb.hpp"
#include "dlug.h"

// Unpacks the tile map stored as bytes in
// `tile_map_bytes` into the `tile_map` structure.
//
// The `tile_map` holds the 'knot' information for each
// of the first N tile variants.  The lowest level arrays
// hold the length of the array as the first entry with
// the subsequent entries alternating between the variant
// and span.  For example:
//
// [idx_0]:
//   [
//     [ n_{idx_0,0}, var_{idx_0,0,0}, span_{idx_0,0,0}, var_{idx_0,0,1}, span_{idx_0,0,1}, ..., var_{idx_0,0,n-1}, span_{idx_0,0,n-1} ],
//     [ n_{idx_0,1}, var_{idx_0,1,0}, span_{idx_0,1,0}, var_{idx_0,1,1}, span_{idx_0,1,1}, ..., var_{idx_0,1,n-1}, span_{idx_0,1,n-1} ],
//   ],
// [idx_1]:
//   [
//     [ n_{idx_1,0}, var_{idx_1,0,0}, span_{idx_1,0,0}, var_{idx_1,0,1}, span_{idx_1,0,1}, ..., var_{idx_1,0,n-1}, span_{idx_1,0,n-1} ],
//     [ n_{idx_1,1}, var_{idx_1,1,0}, span_{idx_1,1,0}, var_{idx_1,1,1}, span_{idx_1,1,1}, ..., var_{idx_1,1,n-1}, span_{idx_1,1,n-1} ],
//   ],
//
// ...
//
// [idx_{N-1}]:
//   [
//     [ n_{idx_{N-1},0}, var_{idx_{N-1},0,0}, span_{idx_{N-1},0,0}, var_{idx_{N-1},0,1}, span_{idx_{N-1},0,1}, ..., var_{idx_{N-1},0,n-1}, span_{idx_{N-1},0,n-1} ],
//     [ n_{idx_{N-1},1}, var_{idx_{N-1},1,0}, span_{idx_{N-1},1,0}, var_{idx_{N-1},1,1}, span_{idx_{N-1},1,1}, ..., var_{idx_{N-1},1,n-1}, span_{idx_{N-1},1,n-1} ],
//   ]
//
//
int cgf_unpack_tile_map(cgf_t *cgf) {
  unsigned char *b;

  int n, dn, N;
  uint32_t n0, n1, v, s, m[2];
  int i, j, k, cur=0;

  N = cgf->tile_map_len;
  b = cgf->tile_map_bytes;

  n = 0;
  while (n<N) {

    dn = dlug_convert_uint32(b + n, &(m[0]));
    if (dn<0) { return -1; }
    n+=dn;

    dn = dlug_convert_uint32(b + n, &(m[1]));
    if (dn<0) { return -1; }
    n+=dn;

    cur++;

    for (j=0; j<2; j++) {
      for (i=0; i<m[j]; i++) {
        dn = dlug_convert_uint32(b + n, &v);
        if (dn<0) { return -1; }
        n+=dn;

        dn = dlug_convert_uint32(b + n, &s);
        if (dn<0) { return -1; }
        n+=dn;
      }
    }
  }

  cgf->n_tile_map = cur;


  cgf->tile_map = (int ***)malloc(sizeof(int **)*cur);
  n = 0;
  cur=0;
  while (n<N) {

    dn = dlug_convert_uint32(b + n, &(m[0]));
    if (dn<0) { return -1; }
    n+=dn;

    dn = dlug_convert_uint32(b + n, &(m[1]));
    if (dn<0) { return -1; }
    n+=dn;

    cgf->tile_map[cur] = (int **)malloc(sizeof(int *)*2);

    cgf->tile_map[cur][0] = (int *)malloc(sizeof(int)*(2*m[0]+1));
    cgf->tile_map[cur][1] = (int *)malloc(sizeof(int)*(2*m[1]+1));

    cgf->tile_map[cur][0][0] = m[0];
    cgf->tile_map[cur][1][0] = m[1];

    for (j=0; j<2; j++) {
      for (i=0; i<m[j]; i++) {
        dn = dlug_convert_uint32(b + n, &v);
        if (dn<0) { return -1; }
        n+=dn;


        dn = dlug_convert_uint32(b + n, &s);
        if (dn<0) { return -1; }
        n+=dn;

        cgf->tile_map[cur][j][2*i+1] = v;
        cgf->tile_map[cur][j][2*i+2] = s;
      }
    }

    cur++;
  }

}

//-----

// http://stackoverflow.com/questions/109023/how-to-count-the-number-of-set-bits-in-a-32-bit-integer
//
// Based on 'divide and merge'.  e.g.
//
// unsigned int count_bit(unsigned int x)
// {
//   x = (x & 0x55555555) + ((x >> 1) & 0x55555555);
//   x = (x & 0x33333333) + ((x >> 2) & 0x33333333);
//   x = (x & 0x0F0F0F0F) + ((x >> 4) & 0x0F0F0F0F);
//   x = (x & 0x00FF00FF) + ((x >> 8) & 0x00FF00FF);
//   x = (x & 0x0000FFFF) + ((x >> 16)& 0x0000FFFF);
//   return x;
// }
//
int NumberOfSetBits(uint32_t u)
{
  u = u - ((u >> 1) & 0x55555555);
  u = (u & 0x33333333) + ((u >> 2) & 0x33333333);
  return (((u + (u >> 4)) & 0x0F0F0F0F) * 0x01010101) >> 24;
}


// This is slower than the above but is more explicit
//
int NumberOfSetBits8(uint8_t u)
{
  u = (u & 0x55) + ((u>>1) & 0x55);
  u = (u & 0x33) + ((u>>2) & 0x33);
  u = (u & 0x0f) + ((u>>4) & 0x0f);
  return u;
}


// Only check the canonical bits for matches within the
// [start_step,start_step+n_step) range.
//
// Store matched results in 'n_match'.
//
int cgf_tile_concordance_0(int *n_match,
    cgf_t *cgf_a, cgf_t *cgf_b,
    int tilepath, int start_step, int n_step) {

  int i, j, k;
  int start_block, end_block, s;
  cgf_path_t *path_a, *path_b;
  uint64_t mask, z;
  uint32_t u32, x32, y32;

  uint64_t start_mask, end_mask;
  int canonical_count=0;

  path_a = &(cgf_a->path[tilepath]);
  path_b = &(cgf_b->path[tilepath]);

  u32 = (0xffffffff >> (start_step%32));
  start_mask = (uint64_t)u32 << 32;

  u32 = (0xffffffff << (32-((start_step+n_step)%32)));
  end_mask = (uint64_t)u32 << 32;

  start_block = start_step / 32;
  end_block = (start_step + n_step) / 32;

  s = start_block;

  x32 = ((path_a->vec[s] & start_mask) >> 32);
  y32 = ((path_b->vec[s] & start_mask) >> 32);
  k = NumberOfSetBits(x32 & y32);
  canonical_count += (32-(start_step%32)) - k;

  for (s=start_block+1; s<end_block; s++) {
    x32 = ((path_a->vec[s] & 0xffffffff00000000 ) >> 32);
    y32 = ((path_b->vec[s] & 0xffffffff00000000 ) >> 32);
    k = NumberOfSetBits(x32 | y32);
    canonical_count += 32 - k;
  }

  if (s==end_block) {
    x32 = ((path_a->vec[s] & end_mask) >> 32);
    y32 = ((path_b->vec[s] & end_mask) >> 32);
    k = NumberOfSetBits(x32 | y32);
    canonical_count += ((start_step+n_step)%32) - k;
  }


  *n_match = canonical_count;

  return 0;
}

// Only consider either canonical tiles or
// cached overflows.  All otherws (final overflows, low quality
// tiles, etc.) will be ignored.
//
int cgf_tile_concordance_1(int *n_match, int *n_ovf,
    cgf_t *cgf_a, cgf_t *cgf_b,
    int tilepath, int start_step, int n_step) {

  int i, j, k, bit_idx;
  int start_block, end_block, s;
  cgf_path_t *path_a, *path_b;
  uint64_t mask, z;
  uint32_t u32, x32, y32, fullx32, fully32;
  uint32_t lx32, ly32;

  uint32_t xor32, and32;

  uint64_t start_mask, end_mask;

  uint8_t hexit_a_n, hexit_a[8], hexit_b_n, hexit_b[8];
  int a_count, b_count;

  int a_ovf_loq = 0, a_ovf_hiq=0, a_ovf_complex=0;
  int b_ovf_loq = 0, b_ovf_hiq=0, b_ovf_complex=0;

  int canon_match_count=0, cache_match_count=0, ovf_count=0;
  int loq_cache_count=0, cache_ovf_count=0;

  unsigned char flag;

  int s_mod, e_mod;
  int skip_beg=0, use_end=32;

  int local_debug = 0;

  path_a = &(cgf_a->path[tilepath]);
  path_b = &(cgf_b->path[tilepath]);

  start_block = start_step / 32;
  end_block = (start_step + n_step) / 32;

  for (s=start_block; s<=end_block; s++) {

    mask = 0xffffffff00000000;
    skip_beg = 0;
    use_end = 32;

    if (s==start_block) {
      u32 = (0xffffffff >> (start_step%32));
      mask &= (uint64_t)u32 << 32;

      skip_beg = start_step % 32;
    }

    if (s==end_block) {

      u32 = (0xffffffff << (32-((start_step+n_step)%32)));
      mask &= (uint64_t)u32 << 32;

      use_end = (start_step + n_step) % 32;
    }

    x32 = ((path_a->vec[s] & mask ) >> 32);
    y32 = ((path_b->vec[s] & mask ) >> 32);
    k = NumberOfSetBits(x32 | y32);
    canon_match_count += (32-skip_beg-(32-use_end)) - k;

    //DEBUG
    if (local_debug) {

      fullx32 = ((path_a->vec[s] & 0xffffffff00000000 ) >> 32);
      fully32 = ((path_b->vec[s] & 0xffffffff00000000 ) >> 32);

      printf(">> s: %i, k: %i, x32: %08x (%08x), y32: %08x (%08x), skip_beg: %d, use_end: %d, mask: %016" PRIx64 "\n",
          s, k,
          (unsigned int)x32, (unsigned int)fullx32,
          (unsigned int)y32, (unsigned int)fully32,
          skip_beg, use_end, mask);
    }

    if (k>0) {

      // need full vector
      //
      x32 = ((path_a->vec[s] & 0xffffffff00000000 ) >> 32);
      y32 = ((path_b->vec[s] & 0xffffffff00000000 ) >> 32);

      hexit_a_n = NumberOfSetBits(x32);
      hexit_b_n = NumberOfSetBits(y32);

      lx32 = path_a->vec[s] & 0xffffffff;
      ly32 = path_b->vec[s] & 0xffffffff;

      //DEBUG
      if (local_debug) {
        printf("  lx32: %08x, ly32: %08x\n", (unsigned int)lx32, (unsigned int)ly32);
      }

      for (i=0; i<8; i++) {
        hexit_a[7-i] = (uint8_t)((lx32 & (0xf << (4*i)))>>(4*i));
        hexit_b[7-i] = (uint8_t)((ly32 & (0xf << (4*i)))>>(4*i));
      }

      a_count=0;
      b_count=0;
      and32 = x32 & y32;

      for (i=31; i>=0; i--) {
        bit_idx = 31-i;

        //DEBUG
        if (local_debug) {
          printf("  [%i(%i)] (%c,%c:%c) a_count %i, b_count %i\n",
              i, bit_idx,
              //i,
              (x32&(1<<i)) ? '*' : '_',
              (y32&(1<<i)) ? '*' : '_',
              (and32&(1<<i)) ? '*' : '_', a_count, b_count);
          if (and32 & (1<<i)) {
            if (a_count<8) { printf("    a[%i]: %x\n", a_count, hexit_a[a_count]); }
            if (b_count<8) { printf("    b[%i]: %x\n", b_count, hexit_b[b_count]); }
          }
        }

        if (and32 & (1<<i)) {
          if ((a_count<8) && (b_count<8) &&
              (hexit_a[a_count] > 0) && (hexit_a[a_count] < 0xd) &&
              (hexit_b[b_count] > 0) && (hexit_b[b_count] < 0xd)) {

            if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
              cache_match_count += ((hexit_a[a_count] == hexit_b[b_count]) ? 1 : 0);

              //DEBUG
              if (local_debug) {
                printf("      cache_match_count++\n");
              }

            }
            else if (local_debug) {
              printf("      skipped (cache_match_count++)\n");
            }


          }
          else {
            flag = 0;

            if (a_count<8) {
              if      (hexit_a[a_count] == 0xe) { a_ovf_loq++; flag |= (1<<0); }
              else if (hexit_a[a_count] == 0xf) { a_ovf_hiq++; flag |= (1<<1); }
              else if (hexit_a[a_count] == 0xd) { a_ovf_complex++; flag |= (1<<2); }
            }

            if (b_count<8) {
              if      (hexit_b[b_count] == 0xe) { b_ovf_loq++; flag |= (1<<3); }
              else if (hexit_b[b_count] == 0xf) { b_ovf_hiq++; flag |= (1<<4); }
              else if (hexit_b[b_count] == 0xd) { b_ovf_complex++; flag |= (1<<5); }
            }

            if ((a_count<8) && (b_count<8)) {
              if (flag & ((1<<0) | (1<<3))) {

                if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
                  loq_cache_count++;

                  //DEBUG
                  if (local_debug) {
                    printf("      loq_cache_count++\n");
                  }

                }
                else if (local_debug) {
                  printf("      skipped (loq_cache_count++)\n");
                }

              }
              else if (flag & ((1<<1) | (1<<4))) {

                if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
                  ovf_count++;

                  //DEBUG
                  if (local_debug) {
                    printf("      ovf_count++\n");
                  }

                }
                else if (local_debug) {
                    printf("      skipped (ovf_count++)\n");
                }

              }
            }
            else {

              if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
                cache_ovf_count++;

                if (local_debug) {
                  printf("        cache_ovf_count++\n");
                }

              }
              else if (local_debug) {
                printf("      skipped (cache_ovf_count++)\n");
              }

            }



          }

        }

        if (x32 & (1<<i)) { a_count++; }
        if (y32 & (1<<i)) { b_count++; }

      }

    }

  }

  *n_match = canon_match_count + cache_match_count;
  *n_ovf = ovf_count;

  return 0;
}

// Still in development
//
int cgf_final_overflow_scan_to_start(cgf_final_overflow_t *fin_ovf, int start_step) {
  int i, j, k, b;
  uint64_t tot_sz, data_sz;
  uint8_t *code, *data;

  tot_sz = fin_ovf->data_record_byte_len;
  data_sz = tot_sz - fin_ovf->data_record_n;

  code = fin_ovf->data_record->code;
  data = fin_ovf->data_record->data;

  //for (b=0; b<data_sz; ) { }

  return 0;
}

/*
// Doesn't work...don't know why
//
int cgf_cache_map_val(uint64_t vec_val, int ofst) {
  int i, count=0;
  unsigned char hx;
  uint32_t u32, mask;

  u32 = (uint32_t)(vec_val>>32);
  mask = (((uint32_t)0xffffffff)>>(31-ofst));

  u32 &= mask;

  count = NumberOfSetBits(u32);

  if (count>=8) { return -1; }

  hx = (unsigned char)((vec_val & (0xf<<(count*4))) >> (count*4));
  return (int)hx;
}
*/

// x
//
int cgf_cache_map_val(uint64_t vec_val, int ofst) {
  int i, count, shft;
  unsigned char hx;
  uint64_t mask, x;
  int local_debug = 0;


  // canonical tile
  if ((vec_val & (((uint64_t)1)<<(32+ofst)))==0) { return -1; }

  if (local_debug) {
    printf("    vec_val %016" PRIx64 ", ofst %i\n", vec_val, ofst);
  }

  //for (i=0, count=-1; i<=ofst; i++) {
  for (i=0, count=0; i<=ofst; i++) {
    if (vec_val & (((uint64_t)1)<<(32+i))) { count++; }
  }

  if (local_debug) {
    printf("    count %i\n", count);
  }

  // overflow
  //
  if (count>8) { return -2; }

  // error, no cache map val at location
  //
  if (count<=0) { return -3; }

  shft=count-1;

  //hx = (unsigned char)((vec_val & (0xf<<(count*4))) >> (count*4));
  //return (int)hx;

  mask = 0xf;
  mask = mask<<(shft*4);
  mask = vec_val & mask;


  if (local_debug) {
    printf("    masked %016" PRIx64 ", count %i\n", vec_val & mask, count);
  }

  mask = mask >> (shft*4);
  hx = (unsigned char)(mask & 0xf);

  // this location is the later portion of a spanning tile
  //
  //if (hx==0) { return -3; }

  return (int)hx;
}

// Count overflow entries from step_start to step_end, inclusive.
// This is needed for determining where in the OverflowMap (or FinalOverflowMap)
// the appropriate entry is.
//
// x
//
int cgf_relative_overflow_count(uint64_t *vec, int step_start, int step_end) {
  int vec_idx, step_off;
  int cur_step, ovf_count=0;
  int cache_map_val;
  uint64_t vec_val;
  uint32_t canon_bits, ovf_bits;

  int local_debug = 0;

  for (cur_step=step_start; cur_step<=step_end; cur_step++) {
    vec_idx = cur_step/32;
    step_off = cur_step%32;

    vec_val = vec[vec_idx];

    if (local_debug) {
      canon_bits = (uint32_t)(vec_val>>32);
      ovf_bits = (uint32_t)(vec_val&0xffffffff);

      printf("  [%i] %08x %08x\n", cur_step, (unsigned int)canon_bits, (unsigned int)ovf_bits);
    }



    cache_map_val = cgf_cache_map_val(vec[vec_idx], step_off);

    if (local_debug) {
      printf("  cgf_relative_overflow_count step %x {%x,%x} cache_map_val %x\n", cur_step, vec_idx, step_off, cache_map_val);
    }

    // error
    //if (cache_map_val==-2) { return -1; }

    // spanning tile
    //
    if (cache_map_val==0) { continue; }

    // valid cache entry
    //
    if ((cache_map_val>0) && (cache_map_val<0xd)) { continue; }

    // complex tiles not implemented, ignore
    //
    if (cache_map_val==0xd) { continue; }

    // canonical tile (no cache map entry), skip
    //
    if (cache_map_val==-1) { continue; }


    // cache map val -2 is overflow of cache,
    // 0xf is hiq overflow, 0xe is loq overflow.
    //
    if ((cache_map_val==0xf) || (cache_map_val==0xe) || (cache_map_val==-2)) {
      ovf_count++;

      if (local_debug) { printf("  > ovf_count++ (--> %d)\n", ovf_count); }

    }

  }

  if (local_debug) {
    printf("  >> cgf_relative_overflow_count step %x {%x,%x} cache_map_val %x, ovf_count %d\n", cur_step, vec_idx, step_off, cache_map_val, ovf_count);
  }


  return ovf_count;
}

// x
//
int is_canonical_tile(uint64_t vec_val, int ofst) {
  return (vec_val) & ( ((uint64_t)1)<<(32+ofst) );
}

// Find variant id of tilepath.tilestep in structure.
// First determine whether it's a canonical tile or
// resides in the cache and if it is, return the value.
// Otherwise, start looking in the overflow and final
// overflow structures.
//
// x
//
int cgf_map_variant_id(cgf_t *cgf, int tilepath, int step) {
  int i, j, k, dn;
  uint64_t nblock, stride, byte_tot;
  uint32_t u32;
  int byte_offset=0;
  int map_skip_count;

  int local_debug = 0;

  cgf_path_t *path;
  cgf_overflow_t *ovf;

  path = &(cgf->path[tilepath]);

  if (is_canonical_tile(path->vec[step/32], (step%32))) { return 0; }

  k = cgf_cache_map_val(path->vec[step/32], step%32);

  // canonical tile
  //
  if (k==-1) { return 0; }

  if ((k>=0) && (k<0xd)) {

    if (local_debug) {
      printf("  cgf_map_variant_id %x.%x got cache %x\n", tilepath, step, k);
    }

    // trailing spanning tile
    //
    if (k==0) { return -1; }

    return k;
  }


  // complex tiles not supported
  //
  if (k==0xd) { return -2; }

  ovf = path->overflow;
  nblock = (ovf->length + ovf->stride - 1) / ovf->stride;
  stride = ovf->stride;

  byte_tot = ovf->map_byte_count;

  for (k=0; k<nblock; k++) {
    if (step < ovf->position[k]) { break; }
  }
  k--;

  if (local_debug) {
    printf("k block %i (step %d (%x), position[%d] %d (%x))\n", k, step, step, k, (int)ovf->position[k], (int)ovf->position[k]);
  }

  byte_offset = ovf->offset[k];

  if (local_debug) {
    printf("byte offset %d (%x)\n", (int)byte_offset, (int)byte_offset);
  }

  map_skip_count = cgf_relative_overflow_count(path->vec, ovf->position[k], step);

  if (local_debug) {
    printf("  cgf_map_variant_id %x.%x map_skip_count %d\n", tilepath, step, map_skip_count);
  }

  k = 0;
  while ((k < map_skip_count) && (byte_offset < byte_tot)) {
    dn = dlug_convert_uint32(ovf->map + byte_offset, &u32);
    if (dn<=0) { return -1; }

    if (local_debug) { printf("  map[%d(%x)] %i, k:%d\n", (int)byte_offset, (int)byte_offset, (int)u32, k); }

    byte_offset += dn;

    k++;
  }

  if (local_debug) {
    printf("  cgf_map_variant_id %x.%x mapval %i (skipped %d)\n", tilepath, step, (int)u32, k);
  }

  return (int)u32;

}

int cgf_final_overflow_map0_peel(uint8_t *bytes,
    int *anchor_step, int *n_allele,
    std::vector<int> allele[]) {
    //std::vector<int> *allele) {
  int i, j, k;
  int dn, n=0;
  int vid, span;
  uint32_t u32, len, aa;

  int local_debug = 0;

  if (local_debug) {
    if (allele!=NULL) {
      printf(">>>> %p\n", allele);
      printf("%i\n", (int)allele[0].size());
      printf("%i\n", (int)allele[1].size());

      if (allele[0].size()>0) {
        printf(">>>> 0: %d\n", allele[0][0]);
      }

      if (allele[1].size()>0) {
        printf(">>>> 1: %d\n", allele[1][0]);
      }
    }
  }

  dn = dlug_convert_uint32(bytes + n, &u32);
  if (dn<=0) { return -1; }
  n += dn;

  *anchor_step = (int)u32;

  if (local_debug) {
    printf("  anchor_step %x\n", (int)u32);
  }

  dn = dlug_convert_uint32(bytes + n, &u32);
  if (dn<=0) { return -1; }
  n += dn;

  *n_allele = (int)u32;
  aa = u32;

  if (local_debug) {
    printf("  n_allele %i\n", (int)aa);
  }

  for (i=0; i<aa; i++) {
    dn = dlug_convert_uint32(bytes + n, &u32);
    if (dn<=0) { return -1; }
    n += dn;

    len=u32;

    for (j=0; j<len; j++) {
      dn = dlug_convert_uint32(bytes + n, &u32);
      if (dn<=0) { return -1; }
      n += dn;

      vid = (int)u32;

      dn = dlug_convert_uint32(bytes + n, &u32);
      if (dn<=0) { return -1; }
      n += dn;

      span = (int)u32;

      if (allele!=NULL) {
        allele[i].push_back(vid);
        allele[i].push_back(span);
      }

    }
  }

  return n;
}

// Determine if the tilepath.tilestep for cgf_a and cgf_b match.
//
int cgf_final_overflow_match(cgf_t *cgf_a, cgf_t *cgf_b, int tilepath, int tilestep) {
  int i, j, k;
  uint64_t n_a, n_b, byte_len_a, byte_len_b;
  uint8_t *code_a, *code_b;
  uint8_t *map_a, *map_b;
  int rec_a, rec_b;
  int step_a, step_b;
  int dn;
  int byte_offset_a, byte_offset_b;
  cgf_final_overflow_t *fin_ovf_a, *fin_ovf_b;

  std::vector<int> knot_a[2], knot_b[2];

  int local_debug = 0;

  fin_ovf_a = cgf_a->path[tilepath].final_overflow;
  fin_ovf_b = cgf_b->path[tilepath].final_overflow;

  n_a = fin_ovf_a->data_record_n;
  byte_len_a = fin_ovf_a->data_record_byte_len;

  n_b = fin_ovf_b->data_record_n;
  byte_len_b = fin_ovf_b->data_record_byte_len;

  code_a = fin_ovf_a->data_record->code;
  map_a  = fin_ovf_a->data_record->data;

  code_b = fin_ovf_b->data_record->code;
  map_b  = fin_ovf_b->data_record->data;

  byte_offset_a = 0;
  byte_offset_b = 0;

  rec_a = 0;
  step_a = -1;
  while ((byte_offset_a < byte_len_a) && (step_a < tilestep) && (rec_a < n_a)) {

    knot_a[0].clear();
    knot_a[1].clear();

    if (code_a[rec_a]==0) {
      dn = cgf_final_overflow_map0_peel(map_a + byte_offset_a, &step_a, &k, knot_a);
      if (dn<=0) { return 0; }
      byte_offset_a += dn;

      if (k!=2) { return 0; }
    } else { return 0; }

    rec_a++;
  }

  rec_b=0;
  step_b = -1;
  while ((byte_offset_b < byte_len_b) && (step_b < tilestep) && (rec_b < n_b)) {

    knot_b[0].clear();
    knot_b[1].clear();


    if (code_b[rec_b]==0) {
      dn = cgf_final_overflow_map0_peel(map_b + byte_offset_b, &step_b, &k, knot_b);
      if (dn<=0) { return 0; }
      byte_offset_b += dn;

      if (k!=2) { return 0; }
    } else { return 0; }

    rec_b++;
  }

  //DEBUG XXX
  if (local_debug) {
    printf("fin: %04x.%04x: fin ovf: rec_a %i, rec_b %i (step_a %x, step_b %x)\n", tilepath, tilestep, rec_a, rec_b, step_a, step_b);
  }

  if (step_a!=step_b) { return 0; }

  if (local_debug) {
    printf("%04x.%04x a:", tilepath, tilestep);
    for (i=0; i<2; i++) {
      printf(" [");
      for (j=0; j<knot_a[i].size(); j++) printf(" %x", knot_a[i][j]);
      printf("]");
    }
    printf("\n");

    printf("%04x.%04x b:", tilepath, tilestep);
    for (i=0; i<2; i++) {
      printf(" [");
      for (j=0; j<knot_b[i].size(); j++) printf(" %x", knot_b[i][j]);
      printf("]");
    }
    printf("\n");

  }

  for (i=0; i<2; i++) {
    if (knot_a[i].size() != knot_b[i].size()) { return 0; }
    for (j=0; j<knot_a[i].size(); j++) {
      if (knot_a[i][j] != knot_b[i][j]) { return 0; }
    }
  }

  //DEBUG XXX
  if (local_debug) {
    printf("fin_ovf++ %04x.%04x\n", tilepath, tilestep);
  }

  return 1;
}

// ovf_step has [ step , codea, code b ]
// where codeX is -1 for overflow, -2 for complex and has the code otherwise.
//
int cgf_overflow_concordance(int *n_match,
    //int *n_fin_ovf,
    cgf_t *cgf_a, cgf_t *cgf_b,
    int tilepath,
    std::vector<int> &ovf_step) {
  int i, j, k, idx;
  int var_a, var_b, step;
  std::vector<int> fin_ovf_step;
  int match_count=0, fin_ovf_count=0;

  int local_debug = 0;

  for (idx=0; idx<ovf_step.size(); idx+=3) {
    step = ovf_step[idx];
    var_a = ovf_step[idx+1];
    var_b = ovf_step[idx+2];

    // complex, ignore
    //
    if ((var_a<-1) || (var_b<-1)) {

      if (local_debug) {
        printf("idx: %d, %x.%x, var_a %d, var_b %d, complex, ignoring\n",
            idx, tilepath, step, var_a, var_b);
      }

      continue;
    }

    if (var_a<0) {
      var_a = cgf_map_variant_id(cgf_a, tilepath, step);
    }

    if (var_b<0) {
      var_b = cgf_map_variant_id(cgf_b, tilepath, step);
    }

    if (local_debug) {
      printf("%x.%x var_a %d, var_b %d\n", tilepath, step, var_a, var_b);
    }

    if ((var_a < 1024) && (var_b < 1024)) {
      if (var_a==var_b) {

        //DEBUG
        if (local_debug) {
          printf("%04x.%04x, var_a %d, var_b %d, ovf_conf++\n", tilepath, step, var_a, var_b);
          printf("mo: %04x.00.%04x\n", tilepath, step);
        }

        match_count++;
      }
    //} else if ((var_a>=1024) && (var_b>=1024)) {
    } else if ((var_a>1024) && (var_b>1024)) {

      // 1024 is a spanning tile, 1025 is a final overflow
      //
      fin_ovf_step.push_back(step);
      fin_ovf_count++;

      //DEBUG
      if (local_debug) {
        printf("%04x.%04x: fin_ovf queue\n", tilepath, step);
      }
    }

  }

  for (i=0; i<fin_ovf_step.size(); i++) {

    if (cgf_final_overflow_match(cgf_a, cgf_b, tilepath, fin_ovf_step[i])) {

      if (local_debug) {
        printf("%04x.%04x: fin_ovf_count++\n", tilepath, fin_ovf_step[i]);
        printf("mf: %04x.00.%04x\n", tilepath, fin_ovf_step[i]);
      }


      match_count++;
    }
  }

  *n_match = match_count;

  return 0;
}

uint8_t cgf_loq_tile(cgf_t *cgf, int tilepath, int tilestep) {
  //uint8_t u8, z8;
  //u8 = cgf->path[tilepath].loq_info->loq_flag[tilestep/8];
  //z8 = (1<<(tilestep%8));
  return cgf->path[tilepath].loq_info->loq_flag[tilestep/8] & (1<<(tilestep%8));
}

// Only consider either canonical tiles,
// cached overflows or tile mapped overflows.
// All otherws (final overflows, low quality
// tiles, etc.) will be ignored.
//
int cgf_tile_concordance_2(int *n_match,
    //int *n_ovf,
    int *n_loq,
    cgf_t *cgf_a, cgf_t *cgf_b,
    int tilepath, int start_step, int n_step) {

  int i, j, k, bit_idx;
  int start_block, end_block, s, s_beg, s_end;
  cgf_path_t *path_a, *path_b;
  uint64_t mask, z;
  uint32_t u32, x32, y32, fullx32, fully32;
  uint32_t lx32, ly32;

  uint32_t xor32, and32;

  uint64_t start_mask, end_mask;

  uint8_t hexit_a_n, hexit_a[8], hexit_b_n, hexit_b[8];
  int a_count, b_count;

  int a_ovf_loq = 0, a_ovf_hiq=0, a_ovf_complex=0;
  int b_ovf_loq = 0, b_ovf_hiq=0, b_ovf_complex=0;

  int canon_match_count=0, cache_match_count=0, ovf_count=0;
  int loq_cache_count=0, cache_ovf_count=0;

  unsigned char flag;

  int s_mod, e_mod;
  int skip_beg=0, use_end=32;

  int local_debug = 0;
  int loq_count=0;

  uint8_t *loq_flag_a, *loq_flag_b;
  uint8_t mask8;

  int ii;
  uint32_t debug32;
  uint64_t debug64;

  std::vector<int> ovf_info;

  path_a = &(cgf_a->path[tilepath]);
  path_b = &(cgf_b->path[tilepath]);

  s_beg = start_step/8;
  s_end = (start_step+n_step+7)/8;
  for (s=s_beg; s<s_end; s++) {
    mask8 = 0xff;
    if (s==s_beg) {
      mask8 = 0xff >> (start_step%8);
    }

    if (s==(s_end-1)) {
      k = (start_step + n_step)%8;
      mask8 &= 0xff << (7-k);
    }

    loq_count += NumberOfSetBits8(mask8 & (path_a->loq_info->loq_flag[s] | path_b->loq_info->loq_flag[s]));
  }
  *n_loq = loq_count;

  start_block = start_step / 32;
  end_block = (start_step + n_step) / 32;

  //loq_flag_a = cgf_a->loq_info->loq_flag;
  //loq_flag_b = cgf_b->loq_info->loq_flag;

  for (s=start_block; s<=end_block; s++) {

    mask = 0xffffffff00000000;
    skip_beg = 0;
    use_end = 32;

    if (s==start_block) {

      //lsb in upper 4 bytes is first entry
      //
      //u32 = (0xffffffff >> (start_step%32));
      u32 = (((uint32_t)(0xffffffff)) << (start_step%32));
      mask &= (uint64_t)u32 << 32;

      skip_beg = start_step % 32;
    }

    if (s==end_block) {

      //lsb in upper 4 bytes is first entry
      //
      //u32 = (0xffffffff << (32-((start_step+n_step)%32)));
      u32 = (((uint32_t)0xffffffff) >> (32-((start_step+n_step)%32)));
      mask &= (uint64_t)u32 << 32;

      use_end = (start_step + n_step) % 32;
      //use_end = 32 - ((start_step + n_step) % 32);
    }

    x32 = ((path_a->vec[s] & mask ) >> 32);
    y32 = ((path_b->vec[s] & mask ) >> 32);
    k = NumberOfSetBits(x32 | y32);
    canon_match_count += (32-skip_beg-(32-use_end)) - k;

    //DEBUG
    if (local_debug) {
      debug32 = x32 | y32;
      for (ii=skip_beg; ii<use_end; ii++) {
        if ((debug32 & (1<<ii))==0) {

          if (s==end_block) {
            printf("?? s: %i, skip_beg %i, use_end %i, debug32 %08x mask %016" PRIx64 "\n", s, skip_beg, use_end, (unsigned int)debug32, mask);
          }

          printf("mc: %04x.00.%04x\n", tilepath, 32*s + ii);
        }
      }
    }

    //DEBUG
    if (local_debug) {
      fullx32 = ((path_a->vec[s] & 0xffffffff00000000 ) >> 32);
      fully32 = ((path_b->vec[s] & 0xffffffff00000000 ) >> 32);

      printf(">> s: %i, k: %i, x32: %08x (%08x), y32: %08x (%08x), skip_beg: %d, use_end: %d, mask: %016" PRIx64 "\n",
          s, k,
          (unsigned int)x32, (unsigned int)fullx32,
          (unsigned int)y32, (unsigned int)fully32,
          skip_beg, use_end, mask);
    }

    if (k>0) {

      // need full vector
      //
      x32 = ((path_a->vec[s] & 0xffffffff00000000 ) >> 32);
      y32 = ((path_b->vec[s] & 0xffffffff00000000 ) >> 32);

      hexit_a_n = NumberOfSetBits(x32);
      hexit_b_n = NumberOfSetBits(y32);

      lx32 = path_a->vec[s] & 0xffffffff;
      ly32 = path_b->vec[s] & 0xffffffff;

      //DEBUG
      if (local_debug) { printf("  lx32: %08x, ly32: %08x\n", (unsigned int)lx32, (unsigned int)ly32); }

      for (i=0; i<8; i++) {
        //hexit_a[7-i] = (uint8_t)((lx32 & (0xf << (4*i)))>>(4*i));
        //hexit_b[7-i] = (uint8_t)((ly32 & (0xf << (4*i)))>>(4*i));
        hexit_a[i] = (uint8_t)((lx32 & (0xf << (4*i)))>>(4*i));
        hexit_b[i] = (uint8_t)((ly32 & (0xf << (4*i)))>>(4*i));
      }

      a_count=0;
      b_count=0;
      and32 = x32 & y32;

      //for (i=31; i>=0; i--) {
      for (i=0; i<32; i++) {
        //bit_idx = 31-i;
        bit_idx = i;

        //DEBUG
        if (local_debug) {
          printf("  [%i(%i)] (%c,%c:%c) a_count %i, b_count %i\n",
              i, bit_idx,
              //i,
              (x32&(1<<i)) ? '*' : '_',
              (y32&(1<<i)) ? '*' : '_',
              (and32&(1<<i)) ? '*' : '_', a_count, b_count);
          if (and32 & (1<<i)) {
            if (a_count<8) { printf("    a[%i]: %x\n", a_count, hexit_a[a_count]); }
            if (b_count<8) { printf("    b[%i]: %x\n", b_count, hexit_b[b_count]); }
          }
        }

        if (and32 & (1<<i)) {
          if ((a_count<8) && (b_count<8) &&
              (hexit_a[a_count] > 0) && (hexit_a[a_count] < 0xd) &&
              (hexit_b[b_count] > 0) && (hexit_b[b_count] < 0xd)) {

            if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
              cache_match_count += ((hexit_a[a_count] == hexit_b[b_count]) ? 1 : 0);

              //DEBUG XXX
              if (local_debug) {
                if (hexit_a[a_count]==hexit_b[b_count]) {
                  printf("mh: %04x.00.%04x\n", tilepath, s*32 + i);
                }
              }

              //DEBUG
              if (local_debug) { printf("      cache_match_count%s\n", (hexit_a[a_count] == hexit_b[b_count]) ? "++" : ".." ); }
            }
            else if (local_debug) { printf("      skipped (cache_match_count++)\n"); }

          }
          else {
            flag = 0;

            if (a_count<8) {
              if      (hexit_a[a_count] == 0xe) { a_ovf_loq++; flag |= (1<<0); }
              else if (hexit_a[a_count] == 0xf) { a_ovf_hiq++; flag |= (1<<1); }
              else if (hexit_a[a_count] == 0xd) { a_ovf_complex++; flag |= (1<<2); }
            }

            if (b_count<8) {
              if      (hexit_b[b_count] == 0xe) { b_ovf_loq++; flag |= (1<<3); }
              else if (hexit_b[b_count] == 0xf) { b_ovf_hiq++; flag |= (1<<4); }
              else if (hexit_b[b_count] == 0xd) { b_ovf_complex++; flag |= (1<<5); }
            }

            if ((a_count<8) && (b_count<8)) {

              // If either loq flags are set, we discard the pair for
              // our concordance count.
              //
              if (flag & ((1<<0) | (1<<3))) {

                if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
                  loq_cache_count++;

                  //DEBUG
                  if (local_debug) { printf("      loq_cache_count++\n"); }
                }
                else if (local_debug) { printf("      skipped (loq_cache_count++)\n"); }

              }

              // Both are high quiality overflow
              //
              else if ( ((flag & (1<<1))>>1) & ((flag & (1<<4))>>4) ) {

              //else if (flag & ((1<<1) | (1<<4))) {

                if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
                  ovf_count++;

                  // push step into vector for later processing
                  //
                  ovf_info.push_back(s*32 + bit_idx);
                  ovf_info.push_back(-1);
                  ovf_info.push_back(-1);

                  //DEBUG
                  if (local_debug) { printf("      ovf_count++ (hiq ovf)\n"); }
                }
                else if (local_debug) { printf("      skipped (ovf_count++)\n"); }

              }
            }
            else {

              if ((bit_idx >= skip_beg) && (bit_idx < use_end)) {
                cache_ovf_count++;

                if ((!cgf_loq_tile(cgf_a, tilepath, s*32 + bit_idx)) &&
                    (!cgf_loq_tile(cgf_b, tilepath, s*32 + bit_idx))) {

                  // push step and information of variant types into vector for later processing
                  //
                  ovf_info.push_back(s*32 + bit_idx);
                  if (a_count<8) {
                    if ((hexit_a[a_count] > 0) && (hexit_a[a_count] < 0xd)) { ovf_info.push_back(hexit_a[a_count]); }
                    else if (hexit_a[a_count] == 0xf) { ovf_info.push_back(-1); }
                    else { ovf_info.push_back(-2); }
                  } else { ovf_info.push_back(-1); }

                  if (b_count<8) {
                    if ((hexit_b[b_count] > 0) && (hexit_b[b_count] < 0xd)) { ovf_info.push_back(hexit_b[b_count]); }
                    else if (hexit_b[b_count] == 0xf) { ovf_info.push_back(-1); }
                    else { ovf_info.push_back(-2); }
                  } else { ovf_info.push_back(-1); }

                  if (local_debug) {
                    int ii = ovf_info.size();
                    printf("        cache_ovf_count++ (a) [%d: %d %d]\n", ovf_info[ii-3], ovf_info[ii-2], ovf_info[ii-1]);
                  }
                }
                else if (local_debug) {
                  printf("      skipped (step %d %x) (cache_ovf_count++) loq tile %d %d (a)\n",
                      s*32 + bit_idx, s*32 + bit_idx,
                      cgf_loq_tile(cgf_a, tilepath, s*32 + bit_idx),
                      cgf_loq_tile(cgf_b, tilepath, s*32 + bit_idx)
                      );
                }

              }
              else if (local_debug) { printf("      skipped (step %d %x) (cache_ovf_count++) (b)\n", s*32 + bit_idx, s*32 + bit_idx); }

            }

          }

        }

        if (x32 & (1<<i)) { a_count++; }
        if (y32 & (1<<i)) { b_count++; }

      }

    }

  }

  if (local_debug) {
    for (i=0; i<ovf_info.size(); i++) { printf("ovf_info[%i]: %x\n", i, ovf_info[i]); }
  }

  //cgf_overflow_concordance(&k, &j, cgf_a, cgf_b, tilepath, ovf_info);
  cgf_overflow_concordance(&k, cgf_a, cgf_b, tilepath, ovf_info);

  if (local_debug) {
    printf(">>>> overflow match %d\n", k);
    printf(">>>> canon match %i, cache_match %i, overflow %i\n", canon_match_count, cache_match_count, k);
  }



  //*n_match = canon_match_count + cache_match_count;
  *n_match = canon_match_count + cache_match_count + k;
  //*n_ovf = ovf_count;

  return 0;
}

int cgf_final_overflow_step_offset(cgf_t *cgf, int tilepath, int tilestep) {
  uint64_t prev_byte_offset, byte_offset, byte_len, n_rec, data_byte_len;
  int n, dn;
  int cur_step, n_allele, rec;
  uint8_t *code, *data;
  cgf_final_overflow_t *fin_ovf;

  fin_ovf = cgf->path[tilepath].final_overflow;
  byte_len = fin_ovf->data_record_byte_len;
  n_rec = fin_ovf->data_record_n;

  code = fin_ovf->data_record->code;
  data = fin_ovf->data_record->data;

  data_byte_len = byte_len - n_rec;
  rec=0;

  byte_offset=0;
  prev_byte_offset=0;

  while ((byte_offset < data_byte_len) && (cur_step < tilestep)) {

    prev_byte_offset = byte_offset;

    if (code[rec]!=0) { return -3; }

    dn = cgf_final_overflow_map0_peel(data+byte_offset, &cur_step, &n_allele, NULL);
    if (dn<=0) { return -2; }
    byte_offset += (uint64_t)dn;

  }

  if (cur_step!=tilestep) { return -1; }

  //return (int)byte_offset;
  return (int)prev_byte_offset;

}

void test_lvl2(cgf_t *cgf, cgf_t *cgf_b) {
  int i, j, k;
  int pt = 0x9e;
  int st = 0x64d;
  int varid;
  int ofst;
  uint8_t *xbuf;
  int lvl=2;
  int n_loq;
  int n_match;

  int anchor_step=-1, n_allele=-1;
  std::vector<int> allele[2];

  for (i=0; i<20; i++) {
    varid = cgf_map_variant_id(cgf, pt, st+i);
    printf("a: %04x.%04x varid %x\n", pt, st+i, varid);

    if (varid==1024) {
      printf("spanning tile...?\n");
    }
    else if (varid > 1024) {

      printf(">>>>> varid > 1024 (%d), step %x\n", varid, st+i);

      allele[0].clear();
      allele[1].clear();

      ofst = cgf_final_overflow_step_offset(cgf, pt, st+i);

      printf(" got ofst: %i\n", ofst);

      if (ofst<0) { printf("NO\n"); exit(1); }

      xbuf = cgf->path[pt].final_overflow->data_record->data + ofst;

      printf(".... %x %x %x %x %x %x\n", xbuf[0], xbuf[1], xbuf[2], xbuf[3], xbuf[4], xbuf[5]);


      //allele[0].push_back(-1);
      //allele[1].push_back(-1);

      /*
      printf("????");
      for (int ii=0; ii<2; ii++) {
        for (j=0; j<allele[ii].size(); j++) {
          printf(" %d", allele[ii][j]);
        }
        printf("\n");
      }
      */

      cgf_final_overflow_map0_peel(cgf->path[pt].final_overflow->data_record->data + ofst, &anchor_step, &n_allele, allele);

      printf("  >>> anchor_step %x, n_allele %i:", anchor_step, n_allele);
      for (int ii=0; ii<2; ii++) {
        printf("[%i](", ii);
        for (j=0; j<allele[ii].size(); j++) {
          printf(" %x", allele[ii][j]);
        }
        printf(") ");
      }
      printf("\n");

    }

  }

  for (i=0; i<20; i++) {
    varid = cgf_map_variant_id(cgf_b, pt, st+i);
    printf("b: %04x.%04x varid %x\n", pt, st+i, varid);

    if (varid==1024) {
      printf("spanning tile...?\n");
    }
    else if (varid >= 1024) {
      allele[0].clear();
      allele[1].clear();

      ofst = cgf_final_overflow_step_offset(cgf, pt, st+i);

      printf(" got ofst: %i\n", ofst);

      if (ofst<0) { printf("NO\n"); exit(1); }

      cgf_final_overflow_map0_peel(cgf->path[pt].final_overflow->data_record->data + ofst, &anchor_step, &n_allele, allele);

      printf("  >>> anchor_step %x, n_allele %i:", anchor_step, n_allele);

      for (int ii=0; ii<2; ii++) {
        printf("[%i](", ii);
        for (j=0; j<allele[ii].size(); j++) {
          printf(" %x", allele[ii][j]);
        }
        printf(") ");
      }
      printf("\n");


    }

  }

  printf("\n\n\n");

  //DEBUG
  i=0x9e;
  j=0;
  k=0;

  // test cgf_relative_overflow_count
  for (k=0; k<100; k++) {
    printf("%04x.%04x: a: %i\n",
        i, k,
        cgf_relative_overflow_count(cgf->path[i].vec, 0, k));

    varid = cgf_map_variant_id(cgf, i, k);
    printf(">>>>>>>>>>>>>> %04x.%04x: varid %x\n", i, k, varid);
  }

  printf("\n\n\n===========================\n\n\n");

  for (k=0; k<100; k++) {
    printf("%04x.%04x: b: %i\n",
        i, k,
        cgf_relative_overflow_count(cgf_b->path[i].vec, 0, k));

    varid = cgf_map_variant_id(cgf_b, i, k);
    printf(">>>>>>>>>>>>>> %04x.%04x: varid %x\n", i, k, varid);
  }

  printf("\n\n\n===========================\n\n\n");

  exit(1);


  cgf_tile_concordance_2(&n_match, &n_loq,
      cgf, cgf_b,
      i, 0, cgf->path[i].n_tile);
  printf("[%x] level: %i, matched %d (loq %d)\n", i, lvl, n_match, n_loq);
  k+=n_match;
  j+=n_loq;

  /*
  j=0;
  k=0;
  for (i=0; i<cgf->path_count; i++) {
    cgf_tile_concordance_2(&n_match, &n_loq,
        cgf, cgf_b,
        i, 0, cgf->path[i].n_tile);
    printf("[%x] level: %i, matched %d (loq %d)\n", i, lvl, n_match, n_loq);
    k+=n_match;
    j+=n_loq;
  }
  */

  printf("level: %i, match: %i, loq: %d\n", lvl, k, j);

}



int main(int argc, char **argv) {
  int i, j, k;
  FILE *fp=stdin;
  char *input_fn = NULL;
  char ch;
  cgf_t *cgf, *cgf_a, *cgf_b;;
  int debug_print = 0, stats_print=0;

  cgf_t **cgfa;
  int n_cgfa=3;

  int n_match, n_ovf;

  int tilepath= -1, tilestep=-1;
  int tilevariant_flag = 0;

  int n_loq=0, n_tot=0;
  int lvl=0;

  int single_path_concordance=-1;


  while ((ch=getopt(argc, argv, "hvi:DSVp:s:l:C:")) != -1) switch (ch) {
    case 'h':
      show_help();
      exit(0);
      break;

    case 'p':
      tilepath = atoi(optarg);
      break;
    case 's':
      tilestep = atoi(optarg);
      break;

    case 'V':
      tilevariant_flag = 1;
      break;

    case 'l':
      lvl = atoi(optarg);
      break;

    case 'C':
      single_path_concordance = atoi(optarg);
      break;

    case 'D':
      debug_print = 1;
      break;
    case 'S':
      stats_print=1;
      break;
    case 'i':
      input_fn = strdup(optarg);
      break;
    case 'v':
      break;
    default:
      break;
  }


  if (input_fn!=NULL) {
    if (!(fp = fopen(input_fn, "r"))) {
      perror(input_fn);
      show_help();
      exit(1);
    }
  } else if (isatty(fileno(stdin))) {
    show_help();
    exit(1);
  }

  //---
  //
  //cgf = load_cgf(fp);
  cgf = load_cgf_buf(fp);
  if (fp!=stdin) { fclose(fp); }

  cgf_b = load_cgf_fn("data/hu826751-GS03052-DNA_B01.cgf");

  //cgf_b = load_cgf_fn("data/hu0211D6-GS01175-DNA_E02.cgf");

  /*
  //cgf_tile_concordance_0(&k, cgf, cgf_b, 0, 3, 5000);
  cgf_tile_concordance_0(&k, cgf, cgf_b, 1, 3, 10000);
  //cgf_tile_concordance_0(&k, cgf, cgf_b, 3, 3, 5000);
  printf(">>> %d\n", k);
  */

  // testing cgf_tile_concordance_0
  //
  /*
  j=0;
  for (i=0; i<cgf->path_count; i++) {
    cgf_tile_concordance_0(&k, cgf, cgf_b, i, 0, cgf->path[i].n_tile);
    printf(">>> %d\n", k);
    j+=k;
  }

  printf(">>>>> %i\n", j);
  */

  // testing cgf_tile_concordance_1
  //
  /*
  if (!debug_print) {
    cgf_tile_concordance_0(&k, cgf, cgf_b, 1, 5607, 104);

    cgf_tile_concordance_1(&n_match, &n_ovf,
        cgf, cgf_b,
        1, 5607, 104);
        //1, 5607, 5);
        //1, 5600, 100);
    printf("canon_match: %i, n_match: %i, n_ovf: %i\n", k, n_match, n_ovf);
  }
  */

  if (tilevariant_flag) {
    j = cgf_map_variant_id(cgf, tilepath, tilestep);
    printf(">>> %04x.%04x: %d (%x)\n", tilepath, tilestep, j, j);
    exit(0);
  }

  /*
  if (!debug_print) {
    //cgf_tile_concordance_2(&n_match, &n_ovf,
    cgf_tile_concordance_2(&n_match, &n_loq,
        cgf, cgf_b,
        //1, 0, 9000);
        1, 5607, 104);
        //1, 5607, 5);
        //1, 5600, 100);
    //printf("n_match: %i, n_ovf: %i\n", n_match, n_ovf);
    printf("n_match: %i, n_loq: %i\n", n_match, n_loq);
  }
  */

  if (lvl==0) {

    k=0;
    for (i=0; i<cgf->path_count; i++) {
      cgf_tile_concordance_0(&n_match, cgf, cgf_b, i, 0, cgf->path[i].n_tile);
      k+=n_match;
    }

    printf("level: %i, canonical match: %i\n", lvl, k);
  }

  else if (lvl==1) {
    j=0;
    k=0;
    for (i=0; i<cgf->path_count; i++) {
      cgf_tile_concordance_1(&n_match, &n_loq,
          cgf, cgf_b,
          i, 0, cgf->path[i].n_tile);
      //printf(">>> matched %d (loq %d)\n", n_match, n_loq);
      k+=n_match;
      j+=n_loq;
    }

    printf("level: %i, canonical+cache match: %i, loq: %d\n", lvl, k, j);

  }

  else if (lvl==2) {

    if (single_path_concordance != -1) {
      cgf_tile_concordance_2(&n_match, &n_loq,
          cgf, cgf_b,
          single_path_concordance, 0, cgf->path[single_path_concordance].n_tile);
      //match_tot += n_match;
      printf("#[%x] level: %i, matched %d (loq %d)\n", single_path_concordance, lvl, n_match, n_loq);
      printf("%04x %d\n", single_path_concordance, n_match);
    }

    else {

    //int pt=0x9e;
    /*
    k=0;
    j=0;
    n_match=0;
    n_loq=0;

    cgf_tile_concordance_2(&n_match, &n_loq,
        cgf, cgf_b,
        tilepath, 0, cgf->path[tilepath].n_tile);
    printf("[%x] level: %i, matched %d (loq %d)\n", tilepath, lvl, n_match, n_loq);
    k+=n_match;
    j+=n_loq;
    */

      int match_tot = 0;

      for (tilepath=0; tilepath<cgf->path_count; tilepath++) {
        cgf_tile_concordance_2(&n_match, &n_loq,
            cgf, cgf_b,
            tilepath, 0, cgf->path[tilepath].n_tile);
        match_tot += n_match;
        printf("#[%x] level: %i, matched %d (loq %d)\n", tilepath, lvl, n_match, n_loq);
        printf("%04x %d\n", tilepath, n_match);
      }

      printf("#match_tot: %i\n", match_tot);
    }

    /*
    j=0;
    k=0;
    for (i=0; i<cgf->path_count; i++) {
      cgf_tile_concordance_2(&n_match, &n_loq,
          cgf, cgf_b,
          i, 0, cgf->path[i].n_tile);
      printf("[%x] level: %i, matched %d (loq %d)\n", i, lvl, n_match, n_loq);
      k+=n_match;
      j+=n_loq;
    }
    */

    //printf("level: %i, match: %i, loq: %d\n", lvl, k, j);

  }

  /*
  j=0;
  k=0;
  for (i=0; i<cgf->path_count; i++) {
  //for (i=0; i<20; i++) {
    cgf_tile_concordance_2(&n_match, &n_loq,
        cgf, cgf_b,
        i, 0, cgf->path[i].n_tile);
    printf(">>> matched %d (loq %d)\n", n_match, n_loq);
    k+=n_match;
    j+=n_loq;
  }

  printf(">>>>> tot: %i (loq: %d)\n", k, j);
  */

  /*
  k=0;
  j=0;
  for (i=0; i<cgf->path_count; i++) {
    cgf_tile_concordance_1(&n_match, &n_ovf,
        cgf, cgf_b,
        i, 0, cgf->path[i].n_tile);
    printf(">>> %d\n", n_match);
    k+=n_match;
    j+=n_ovf;
  }

  printf(">>>>> x %i (ovf %d)\n", k, j);
  */



  /*
  cgfa = (cgf_t **)malloc(sizeof(cgf_t *)*n_cgfa);
  for (i=0; i<n_cgfa; i++) {
    cgfa[i] = load_cgf_fn(input_fn);
    if (cgfa[i]==NULL) { printf("nope\n"); }
  }
  printf("ok\n");
  */

  if (debug_print) { debug_print_cgf(cgf); }
  if (stats_print) { stats_print_cgf(cgf); }


}
