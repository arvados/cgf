//package main
package cgf

import "fmt"
//import "crypto/md5"

import "github.com/abeconnelly/dlug"

import "github.com/abeconnelly/cglf"

type HeaderIntermediate struct {
  magic [8]byte
  ver string
  libver string
  pathcount int

  TileMap []TileMapEntry
  TileMapBytes []byte

  StepPerPath []int
  path_offset []int

  PathBytes [][]byte
  //pathis []pathintermediate
  pathis []PathIntermediate

}

type PathIntermediate struct {
  name string
  ntile int
  VecUint64 []uint64
  cgfi CGFIntermediate
  ofsi OverflowIntermediate
  fofsi FinalOverflowIntermediate
  loqi LoqIntermediate
}

type CGFIntermediate struct {
  step  [][]int
  seq   [][]string
  varid [][]int
  span  [][]int
  loq   [][]bool
  nocall_start_len [][][]int

  TileMapKey string
  TileMapPos int
  loq_flag bool
  tot_span int
}

type OverflowIntermediate struct {
  recno int
  stride int
  tilepos []int
  TileMap []int

  offset_idx []int
  tilepos_idx []int

  final_overflow_flag []bool
  span_flag []bool
}

type FinalOverflowIntermediate struct {
  recno int
  tilepos []int
  variant_ints []int  // format is { step { nallele { l0 { var + span , ... } } { l1 { var + span ... } } } }
                      // NOTE: l0 and l1 are number of record.  i.e. l0 will be 1 if there is only one var+span combo
}

type VectorElement struct {
  canon_flag bool
  cache_flag bool
  ovf_cache_flag bool
  ovf_flag bool
  fin_ovf_flag bool
  span_flag bool

  loq_flag bool

  knot CGFIntermediate
  hexit_pos int
  vec_pos int
}

type LoqIntermediate struct {
  loqinfo_bytecount int
  count         int
  code          int
  stride        int
  tilepos       []int

  homflag       []bool
  loqinfo_ints  []int

  loq_flag      []bool

  // key is step
  loqi_info     map[int]CGFIntermediate
}


//========================================================================
//========================================================================
//========================================================================
//========================================================================
//========================================================================

func VectorElementOverflowCount(prep_vector []VectorElement, st, n int) int {
  ovf_count := 0
  for i:=st; i<(st+n); i++ {
    if prep_vector[i].ovf_flag || prep_vector[i].ovf_cache_flag {
      ovf_count++
    }
  }
  return ovf_count
}


func _knot_tot_span(knot *CGFIntermediate) int {
  sp := [2]int{}
  for allele:=0; allele<2; allele++ {
    for i:=0; i<len(knot.span[allele]); i++ {
      sp[allele] += knot.span[allele][i]
    }
  }

  max_span := sp[0]
  if sp[1]>sp[0] { max_span = sp[1] }

  knot.tot_span = max_span
  return max_span
}

// return span
//
func _add_knot(knot *CGFIntermediate, allele, step_idx int, ti TileInfo, sglf *cglf.SGLF) (int,error) {
  if len(ti.NocallStartLen)>0 {
    knot.loq[allele] = append(knot.loq[allele], true)
    knot.loq_flag = true
  } else {
    knot.loq[allele] = append(knot.loq[allele], false)
  }

  sglf_info := cglf.SGLFInfo{}
  var ok bool

  // sglf_info only holds a valid path and step
  //
  if step_idx>0 {
    sglf_info,ok = sglf.PfxTagLookup[ti.PfxTag]

  //} else if (ti.SfxTag == "") && (len(sglf.Lib)==1) {
  } else if (ti.PfxTag == "") && (ti.SfxTag == "") {

    found := false
    for p := range sglf.Lib {
      for s := range sglf.Lib[p] {

        for k:=0; k<len(sglf.Lib[p][s]); k++ {
          found = tile_cmp(ti.Seq, sglf.Lib[p][s][k])

          //fmt.Printf(">>> cp p%d, s%d, k%d, found %v\n", p, s, k, found)

          if found {
            ok = true
            sglf_info = sglf.LibInfo[p][s][k]
            break
          }

        }

      }
      if found { break }
    }

  } else {
    sglf_info,ok = sglf.SfxTagLookup[ti.SfxTag]
  }

  if !ok {

    //DEBUG
    //for k,v := range sglf.SfxTagLookup { fmt.Printf(">>> %v %v\n", k, v) }

    return -1,fmt.Errorf("could not find prefix/suffix (%s/%s) in sglf (allele_idx %d, step_idx %d (%x))\n",
      ti.PfxTag, ti.SfxTag, 0, step_idx, step_idx)
  }

  path := sglf_info.Path
  step := sglf_info.Step

  // We need to search for the variant in the Lib to find
  // the rest of the information, including span
  //
  var_idx,e := lookup_variant_index(ti.Seq, sglf.Lib[path][step])
  if e!=nil { return -1,e }

  sglf_info = sglf.LibInfo[path][step][var_idx]
  span := sglf_info.Span

  seq := sglf.Lib[path][step][var_idx]

  knot_allele_idx := len(knot.varid[allele])

  knot.seq[allele] = append(knot.seq[allele], seq)
  knot.varid[allele] = append(knot.varid[allele], var_idx)
  knot.span[allele] = append(knot.span[allele], span)
  knot.step[allele] = append(knot.step[allele], step)

  nc_vec := make([]int,  0, 1024)
  nc_vec = append(nc_vec, ti.NocallStartLen...)
  dummy := [][]int{}
  dummy = append(dummy, []int{})
  knot.nocall_start_len[allele] = append(knot.nocall_start_len[allele], dummy...)
  knot.nocall_start_len[allele][knot_allele_idx] = append(knot.nocall_start_len[allele][knot_allele_idx], nc_vec...)

  return sglf_info.Span,nil
}

func _init_knot(knot *CGFIntermediate) {
  knot.seq = make([][]string, 2)
  knot.varid = make([][]int, 2)
  knot.span = make([][]int, 2)
  knot.step = make([][]int, 2)
  knot.loq = make([][]bool, 2)
  knot.nocall_start_len = make([][][]int, 2)
}


//====   _                    _           _       _                               _ _       _
//====  | |__   ___  __ _  __| | ___ _ __(_)_ __ | |_ ___ _ __ _ __ ___   ___  __| (_) __ _| |_ ___
//====  | '_ \ / _ \/ _` |/ _` |/ _ \ '__| | '_ \| __/ _ \ '__| '_ ` _ \ / _ \/ _` | |/ _` | __/ _ \
//====  | | | |  __/ (_| | (_| |  __/ |  | | | | | ||  __/ |  | | | | | |  __/ (_| | | (_| | ||  __/
//====  |_| |_|\___|\__,_|\__,_|\___|_|  |_|_| |_|\__\___|_|  |_| |_| |_|\___|\__,_|_|\__,_|\__\___|


func HeaderIntermediateCmp(hdri0, hdri1 HeaderIntermediate) error {

  for i:=0; i<8; i++ {
    if hdri0.magic[i] != hdri1.magic[i] {
      return fmt.Errorf("magic byte mismatch at %d (%d != %d)", i, hdri0.magic[i], hdri1.magic[i])
    }
  }

  if hdri0.ver != hdri1.ver {
    return fmt.Errorf("version mismatch (%s != %s)", hdri0.ver, hdri1.ver)
  }

  if hdri0.libver != hdri1.libver {
    return fmt.Errorf("libversion mismatch (%s != %s)", hdri0.libver, hdri1.libver)
  }

  if hdri0.pathcount != hdri1.pathcount {
    return fmt.Errorf("pathcount mismatch (%v != %v)", hdri0.pathcount, hdri1.pathcount)
  }

  if len(hdri0.TileMap)!= len(hdri1.TileMap) {
    return fmt.Errorf("TileMap length mismatch (%v != %v)", len(hdri0.TileMap), len(hdri1.TileMap))
  }

  for i:=0; i<len(hdri0.TileMap); i++ {
    if hdri0.TileMap[i].TileMap != hdri1.TileMap[i].TileMap {
      return fmt.Errorf("TileMap entry %d TileMap mismatch (%v != %v)", i, hdri0.TileMap[i].TileMap, hdri1.TileMap[i].TileMap)
    }

    if len(hdri0.TileMap[i].Variant) != len(hdri1.TileMap[i].Variant) {
      return fmt.Errorf("TileMap Variant length mismatch at %d (%v != %v)", i, len(hdri0.TileMap[i].Variant), len(hdri1.TileMap[i].Variant))
    }

    if len(hdri0.TileMap[i].Span) != len(hdri1.TileMap[i].Span) {
      return fmt.Errorf("TileMap Span length mismatch at %d (%v != %v)", i, len(hdri0.TileMap[i].Span), len(hdri1.TileMap[i].Span))
    }

    for j:=0; j<len(hdri0.TileMap[i].Variant); j++ {
      if len(hdri0.TileMap[i].Variant[j]) != len(hdri1.TileMap[i].Variant[j]) {
        return fmt.Errorf("TileMap Variant length mismatch at %d, %d (%v != %v)", i, j, len(hdri0.TileMap[i].Variant[j]), len(hdri1.TileMap[i].Variant[j]))
      }

      for k:=0; k<len(hdri0.TileMap[i].Variant[j]); k++ {
        if hdri0.TileMap[i].Variant[j][k] != hdri1.TileMap[i].Variant[j][k] {
          return fmt.Errorf("TileMap Variant element mismatch at %d, %d, %d (%v != %v)", i, j, k, (hdri0.TileMap[i].Variant[j][k]), (hdri1.TileMap[i].Variant[j][k]))
        }
      }
    }

    for j:=0; j<len(hdri0.TileMap[i].Span); j++ {
      if len(hdri0.TileMap[i].Span[j]) != len(hdri1.TileMap[i].Span[j]) {
        return fmt.Errorf("TileMap Span length mismatch at %d, %d (%v != %v)", i, j, len(hdri0.TileMap[i].Span[j]), len(hdri1.TileMap[i].Span[j]))
      }

      for k:=0; k<len(hdri0.TileMap[i].Span[j]); k++ {
        if hdri0.TileMap[i].Span[j][k] != hdri1.TileMap[i].Span[j][k] {
          return fmt.Errorf("TileMap Span element mismatch at %d, %d, %d (%v != %v)", i, j, k, (hdri0.TileMap[i].Span[j][k]), (hdri1.TileMap[i].Span[j][k]))
        }
      }
    }

  }

  if len(hdri0.TileMapBytes) != len(hdri1.TileMapBytes) {
    return fmt.Errorf("TileMap bytes lenght mismatch (%v != %v)", len(hdri0.TileMapBytes), len(hdri1.TileMapBytes))
  }

  for i:=0; i<len(hdri0.TileMapBytes); i++ {
    if hdri0.TileMapBytes[i] != hdri1.TileMapBytes[i] {
      return fmt.Errorf("TileMap byte mismatch at %d (%v != %v)", i, (hdri0.TileMapBytes[i]), (hdri1.TileMapBytes[i]))
    }
  }

  if len(hdri0.StepPerPath) != len(hdri1.StepPerPath) {
    return fmt.Errorf("TileMap step_per_byte length mismatch (%v != %v)", len(hdri0.StepPerPath), len(hdri1.StepPerPath))
  }

  for i:=0; i<len(hdri0.StepPerPath); i++ {
    if (hdri0.StepPerPath[i]) != (hdri1.StepPerPath[i]) {
      return fmt.Errorf("TileMap step_per_byte mismatch at %d (%v != %v)", i, (hdri0.StepPerPath[i]), (hdri1.StepPerPath[i]))
    }
  }

  if len(hdri0.path_offset) != len(hdri1.path_offset) {
    return fmt.Errorf("TileMap step_per_byte length mismatch (%v != %v)", len(hdri0.path_offset), len(hdri1.path_offset))
  }

  for i:=0; i<len(hdri0.path_offset); i++ {
    if (hdri0.path_offset[i]) != (hdri1.path_offset[i]) {
      return fmt.Errorf("TileMap step_per_byte mismatch at %d (%v != %v)", i, (hdri0.path_offset[i]), (hdri1.path_offset[i]))
    }
  }

  if hdri0.pathcount != len(hdri0.StepPerPath) {
    return fmt.Errorf("sanity: pathcount %d does not match StepPerPath %d", hdri0.pathcount, len(hdri0.StepPerPath))
  }

  if (hdri0.pathcount+1) != len(hdri0.path_offset) {
    return fmt.Errorf("sanity: pathcount %d does not match path_offset %d", hdri0.pathcount, len(hdri0.path_offset))
  }

  return nil
}

func HeaderIntermediateFromBytes(b []byte) (HeaderIntermediate,int) {
  hdri := HeaderIntermediate{}
  var dummy uint64
  var dn int

  n:=0

  for i:=0; i<8; i++ { hdri.magic[i] = b[n+i] }
  n+=8

  dummy,dn = dlug.ConvertUint64(b[n:])
  n+=dn

  ns := int(dummy)


  hdri.ver = string(b[n:n+ns])
  n+=ns

  dummy,dn = dlug.ConvertUint64(b[n:])
  n+=dn

  ns = int(dummy)

  hdri.libver = string(b[n:n+ns])
  n+=ns

  dummy = byte2uint64(b[n:n+8])
  n+=8

  hdri.pathcount = int(dummy)

  dummy = byte2uint64(b[n:n+8])
  n+=8

  tilemaplen := int(dummy)

  hdri.TileMapBytes = b[n:n+tilemaplen]
  hdri.TileMap = UnpackTileMap(b[n:n+tilemaplen])
  n += tilemaplen

  for i:=0; i<hdri.pathcount; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8
    hdri.StepPerPath = append(hdri.StepPerPath, int(dummy))
  }

  for i:=0; i<=hdri.pathcount; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8
    hdri.path_offset = append(hdri.path_offset, int(dummy))
  }

  hdri.PathBytes = make([][]byte, hdri.pathcount)

  PathBytes := b[n:]
  for i:=1; i<=hdri.pathcount; i++  {
    dn := int(hdri.path_offset[i] - hdri.path_offset[i-1])
    if dn==0 { continue }

    z:=hdri.path_offset[i-1] ; _ = z
    hdri.PathBytes[i-1] = PathBytes[hdri.path_offset[i-1]:hdri.path_offset[i-1]+dn]
  }

  return hdri,n
}

func BytesFromHeaderIntermediate(hdri HeaderIntermediate) []byte {
  buf := make([]byte, 64)
  b := make([]byte, 0, 1024)

  b = append(b, hdri.magic[:]...)

  mbytes := dlug.MarshalUint64(uint64(len(hdri.ver)))
  b = append(b, mbytes...)

  s := []byte(hdri.ver)
  b = append(b, s...)

  mbytes = dlug.MarshalUint64(uint64(len(hdri.libver)))
  b = append(b, mbytes...)

  s = []byte(hdri.libver)
  b = append(b, s...)

  tobyte64(buf, uint64(hdri.pathcount))
  b = append(b, buf[0:8]...)

  tobyte64(buf, uint64(len(hdri.TileMapBytes)))
  b = append(b, buf[0:8]...)

  b = append(b, hdri.TileMapBytes...)

  for i:=0; i<len(hdri.StepPerPath); i++ {
    tobyte64(buf, uint64(hdri.StepPerPath[i]))
    b = append(b, buf[0:8]...)
  }

  if len(hdri.path_offset)==0 {
    tobyte64(buf, uint64(0))
    b = append(b, buf[0:8]...)
  } else {
    for i:=0; i<len(hdri.path_offset); i++ {
      tobyte64(buf, uint64(hdri.path_offset[i]))
      b = append(b, buf[0:8]...)
    }
  }

  return b
}

//====                        __ _               _       _                               _ _       _
//====    _____   _____ _ __ / _| | _____      _(_)_ __ | |_ ___ _ __ _ __ ___   ___  __| (_) __ _| |_ ___
//====   / _ \ \ / / _ \ '__| |_| |/ _ \ \ /\ / / | '_ \| __/ _ \ '__| '_ ` _ \ / _ \/ _` | |/ _` | __/ _ \
//====  | (_) \ V /  __/ |  |  _| | (_) \ V  V /| | | | | ||  __/ |  | | | | | |  __/ (_| | | (_| | ||  __/
//====   \___/ \_/ \___|_|  |_| |_|\___/ \_/\_/ |_|_| |_|\__\___|_|  |_| |_| |_|\___|\__,_|_|\__,_|\__\___|
//====

func OverflowIntermediateCmp(ofsi0, ofsi1 OverflowIntermediate) error {
  if ofsi0.stride != ofsi1.stride { return fmt.Errorf("stride mismatch") }
  if len(ofsi0.tilepos) != len(ofsi1.tilepos) { return fmt.Errorf("tilepos length mismatch") }
  if len(ofsi0.TileMap) != len(ofsi1.TileMap) { return fmt.Errorf("TileMap length mismatch") }
  if len(ofsi0.final_overflow_flag) != len(ofsi1.final_overflow_flag) { return fmt.Errorf("final_overflow_flag mismatch") }
  if len(ofsi0.span_flag) != len(ofsi1.span_flag) {
    return fmt.Errorf("span_flag mismatch (%v != %v)", len(ofsi0.span_flag), len(ofsi1.span_flag))
  }

  for i:=0; i<len(ofsi0.TileMap); i++ {
    if ofsi0.TileMap[i] != ofsi1.TileMap[i] {
      return fmt.Errorf( fmt.Sprintf("TileMap mismatch at %d, %d != %d", i, ofsi0.TileMap[i], ofsi1.TileMap[i]) )
    }
  }

  for i:=0; i<len(ofsi0.final_overflow_flag); i++ {
    if ofsi0.final_overflow_flag[i] != ofsi1.final_overflow_flag[i] {
      return fmt.Errorf("final_overflow_flag mismatch at %d, %v != %v", i, ofsi0.final_overflow_flag[i], ofsi1.final_overflow_flag[i])
    }
  }

  for i:=0; i<len(ofsi0.span_flag); i++ {
    if ofsi0.span_flag[i] != ofsi1.span_flag[i] {
      return fmt.Errorf("span_flag mismatch at %d, %v != %v", i, ofsi0.span_flag[i], ofsi1.span_flag[i])
    }
  }

  return nil
}

func OverflowIntermediateFromBytes(b []byte) (OverflowIntermediate,int) {
  ofsi := OverflowIntermediate{}

  var dummy uint64
  var dn int

  n:=0

  // Length
  //
  dummy = byte2uint64(b[n:n+8])
  n+=8

  NRec := dummy
  ofsi.recno = int(NRec)

  // Stride
  //
  dummy = byte2uint64(b[n:n+8])
  n+=8

  stride := dummy
  ofsi.stride = int(stride)

  // MapByteCount
  //
  dummy = byte2uint64(b[n:n+8])
  n+=8

  mapbytecount := int(dummy)

  ofs_len := int((NRec+stride-1)/stride)

  ofsi.final_overflow_flag = make([]bool, 0, 1024)
  ofsi.span_flag = make([]bool, 0, 1024)
  ofsi.tilepos_idx = make([]int, 0, 1024)

  ofsi.offset_idx = make([]int, 0, 1024)
  for i:=0; i<ofs_len; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8

    ofsi.offset_idx = append(ofsi.offset_idx, int(dummy))
  }

  ofsi.tilepos_idx = make([]int, 0, 1024)
  for i:=0; i<ofs_len; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8

    ofsi.tilepos_idx = append(ofsi.tilepos_idx, int(dummy))
  }

  for i:=0; i<int(NRec); i++ {
    ofsi.tilepos = append(ofsi.tilepos, ofsi.tilepos_idx[i/int(stride)])
  }

  read_rec := 0
  map_byte_pos:=0
  for map_byte_pos < mapbytecount {
    dummy,dn = dlug.ConvertUint64(b[n:])
    map_byte_pos += dn
    n+=dn


    is_span := false
    if dummy == 1024 {
      dummy = 0
      is_span = true
    }

    is_fin_ovf := false
    if dummy == 1025 {
      dummy = 0
      is_fin_ovf = true
    }

    ofsi.TileMap = append(ofsi.TileMap, int(dummy))

    ofsi.final_overflow_flag = append(ofsi.final_overflow_flag, is_fin_ovf)
    ofsi.span_flag = append(ofsi.span_flag, is_span)

    read_rec++

  }

  return ofsi,n
}

func BytesFromOverflowIntermediate(ofsi OverflowIntermediate) []byte {
  buf := make([]byte, 64)
  offset_bytes := make([]byte, 0, 1024)

  // number of records
  //
  tobyte64(buf, uint64(len(ofsi.tilepos)))
  offset_bytes = append(offset_bytes, buf[0:8]...)

  // stride
  //
  tobyte64(buf, uint64(ofsi.stride))
  offset_bytes = append(offset_bytes, buf[0:8]...)

  offset_idx := make([]uint64, 0, 1024)
  tilepos_idx := make([]uint64, 0, 1024)

  // construct map bytes, record offset and tilepos index
  // along the way.
  //
  map_bytes := make([]byte, 0, 1024)
  for i:=0; i<len(ofsi.tilepos); i++ {
    if i%256 == 0 {
      offset_idx = append(offset_idx, uint64(len(map_bytes)))
      tilepos_idx = append(tilepos_idx, uint64(ofsi.tilepos[i]))
    }

    val := ofsi.TileMap[i]
    if ofsi.span_flag[i] { val = 1024 }
    if ofsi.final_overflow_flag[i] { val = 1025 }

    mbytes := dlug.MarshalUint64(uint64(val))
    map_bytes = append(map_bytes, mbytes...)
  }

  // MapByteCount
  //
  tobyte64(buf, uint64(len(map_bytes)))
  offset_bytes = append(offset_bytes, buf[0:8]...)

  // offset
  //
  for i:=0; i<len(offset_idx); i++ {
    tobyte64(buf, offset_idx[i])
    offset_bytes = append(offset_bytes, buf[0:8]...)
  }

  // tilepos
  //
  for i:=0; i<len(tilepos_idx); i++ {
    tobyte64(buf, tilepos_idx[i])
    offset_bytes = append(offset_bytes, buf[0:8]...)
  }

  // map
  //
  offset_bytes = append(offset_bytes, map_bytes...)

  return offset_bytes

}

func (ctx *CGFContext) ConstructOffsetIntermediate(prep_vector []VectorElement) OverflowIntermediate {
  ofsi := OverflowIntermediate{}

  ofsi.tilepos = make([]int, 0, 1024)
  ofsi.TileMap = make([]int, 0, 1024)

  ofsi.offset_idx = make([]int, 0, 1024)
  ofsi.tilepos_idx = make([]int, 0, 1024)

  ofsi.final_overflow_flag = make([]bool, 0, 1024)
  ofsi.span_flag = make([]bool, 0, 1024)

  for i:=0; i<len(prep_vector); i++ {
    if prep_vector[i].canon_flag { continue }
    if prep_vector[i].ovf_flag {
      ofsi.tilepos = append(ofsi.tilepos, i)
      ofsi.TileMap = append(ofsi.TileMap, prep_vector[i].knot.TileMapPos)

      tf := prep_vector[i].fin_ovf_flag
      //tf := false
      //if prep_vector[i].knot.tilemap_pos > 1023 { tf = true }
      ofsi.final_overflow_flag = append(ofsi.final_overflow_flag, tf)

      tf = false
      if prep_vector[i].span_flag { tf = true }
      ofsi.span_flag = append(ofsi.span_flag, tf)

    }
  }

  ofsi.stride = 256

  return ofsi
}


//========================================================================
//========================================================================
//========================================================================
//========================================================================
//========================================================================

//====    __ _             _                      __ _               _       _                               _ _       _
//====   / _(_)_ __   __ _| | _____   _____ _ __ / _| | _____      _(_)_ __ | |_ ___ _ __ _ __ ___   ___  __| (_) __ _| |_ ___
//====  | |_| | '_ \ / _` | |/ _ \ \ / / _ \ '__| |_| |/ _ \ \ /\ / / | '_ \| __/ _ \ '__| '_ ` _ \ / _ \/ _` | |/ _` | __/ _ \
//====  |  _| | | | | (_| | | (_) \ V /  __/ |  |  _| | (_) \ V  V /| | | | | ||  __/ |  | | | | | |  __/ (_| | | (_| | ||  __/
//====  |_| |_|_| |_|\__,_|_|\___/ \_/ \___|_|  |_| |_|\___/ \_/\_/ |_|_| |_|\__\___|_|  |_| |_| |_|\___|\__,_|_|\__,_|\__\___|
//====

//func finaloverflowintermediate_cmp(fofsi0, fofsi1 finaloverflowintermediate) error {
func FinalOverflowIntermediateCmp(fofsi0, fofsi1 FinalOverflowIntermediate) error {
  if len(fofsi0.tilepos)!=len(fofsi1.tilepos) {
    return fmt.Errorf( fmt.Sprintf("tilepos length mismatch: %d != %d", len(fofsi0.tilepos), len(fofsi1.tilepos)) )
  }

  if len(fofsi0.variant_ints) != len(fofsi1.variant_ints) {
    return fmt.Errorf( fmt.Sprintf("variant_ints length mismatch: %d != %d", len(fofsi0.variant_ints), len(fofsi1.variant_ints)) )
  }

  for i:=0; i<len(fofsi0.variant_ints); i++ {
    if fofsi0.variant_ints[i] != fofsi1.variant_ints[i] {
      return fmt.Errorf( fmt.Sprintf("variant_ints mismatch at %d: %d != %d", i, fofsi0.variant_ints[i], fofsi1.variant_ints[i]) )
    }
  }

  return nil
}

func FinalOverflowIntermediateFromBytes(b []byte) (FinalOverflowIntermediate,int) {
  fofsi := FinalOverflowIntermediate{}

  fofsi.tilepos = make([]int, 0, 1024)

  var dummy uint64
  var dn int

  n:=0

  // nrecord
  //
  dummy = byte2uint64(b[n:n+8])
  n += 8

  nrec := int(dummy)

  // data record byte length
  //
  dummy = byte2uint64(b[n:n+8])
  n += 8

  bytelen := int(dummy)

  code := make([]byte, nrec)

  for i:=0 ;i<nrec; i++ {
    code[i] = b[n+i]
  }
  n+=nrec

  pos:=nrec
  for pos<bytelen {
    dummy,dn = dlug.ConvertUint64(b[n:])
    n+=dn
    pos += dn


    tilestep := int(dummy)
    fofsi.variant_ints = append(fofsi.variant_ints, tilestep)
    fofsi.tilepos = append(fofsi.tilepos, tilestep)

    dummy,dn = dlug.ConvertUint64(b[n:])
    n+=dn
    pos += dn

    nallele := int(dummy)
    fofsi.variant_ints = append(fofsi.variant_ints, nallele)

    for i:=0; i<nallele; i++ {
      dummy,dn = dlug.ConvertUint64(b[n:])
      n+=dn
      pos += dn

      len_allele_knot := int(dummy)
      fofsi.variant_ints = append(fofsi.variant_ints, len_allele_knot)

      for j:=0; j<len_allele_knot; j++ {

        dummy,dn = dlug.ConvertUint64(b[n:])
        n+=dn
        pos += dn

        var_id := int(dummy)
        fofsi.variant_ints = append(fofsi.variant_ints, var_id)

        dummy,dn = dlug.ConvertUint64(b[n:])
        n+=dn
        pos += dn

        span := int(dummy)
        fofsi.variant_ints = append(fofsi.variant_ints, span)

      }

    }

  }

  return fofsi,n

}

func BytesFromFinalOverflowIntermediate(fofsi FinalOverflowIntermediate) []byte {
  buf := make([]byte, 64)
  fof_bytes := make([]byte, 0, 1024)

  // Number of records
  //
  tobyte64(buf, uint64(len(fofsi.tilepos)))
  fof_bytes = append(fof_bytes, buf[0:8]...)

  // redundant...
  //
  code := make([]byte, len(fofsi.tilepos))
  for i:=0; i<len(code); i++ { code[i] = 0 }

  data_bytes := make([]byte, 0, 1024)
  for i:=0; i<len(fofsi.variant_ints); i++ {
    vbytes := dlug.MarshalUint64(uint64(fofsi.variant_ints[i]))
    data_bytes = append(data_bytes, vbytes...)
  }

  // byte length of data record
  //
  bytecount := uint64(len(code) + len(data_bytes))
  tobyte64(buf, bytecount)
  fof_bytes = append(fof_bytes, buf[0:8]...)

  // code section
  //
  fof_bytes = append(fof_bytes, code...)

  // data records
  //
  fof_bytes = append(fof_bytes, data_bytes...)

  return fof_bytes
}

func (ctx *CGFContext) ConstructFinalOffsetIntermediate(prep_vector []VectorElement) FinalOverflowIntermediate {
  fofsi := FinalOverflowIntermediate{}

  fofsi.tilepos = make([]int, 0, 1024)
  fofsi.variant_ints = make([]int, 0, 1024)

  for i:=0; i<len(prep_vector); i++ {
    if prep_vector[i].fin_ovf_flag {
      fofsi.tilepos = append(fofsi.tilepos, i)

      knot := prep_vector[i].knot
      fofsi.variant_ints = append(fofsi.variant_ints,i)
      fofsi.variant_ints = append(fofsi.variant_ints, 2)
      for allele:=0 ; allele<2; allele++ {
        fofsi.variant_ints = append(fofsi.variant_ints, len(knot.varid[allele]))
        for i:=0; i<len(knot.varid[allele]); i++ {
          fofsi.variant_ints = append(fofsi.variant_ints, knot.varid[allele][i])
          fofsi.variant_ints = append(fofsi.variant_ints, knot.span[allele][i])
        }
      }

    }
  }

  return fofsi
}

func (ctx *CGFContext) ConstructUint64Vector(prep_vector []VectorElement) []uint64 {

  ret_vec := make([]uint64, 0, 1024)

  for i:=0; i<len(prep_vector); i+=32 {
    var cur_v uint64

    m := 32

    if i+32 > len(prep_vector) { m = len(prep_vector)%32 }

    for j:=0; j<m; j++ {

      if prep_vector[i+j].canon_flag { continue }

      cur_v |= (1<<(32+uint(j)))

      if prep_vector[i+j].cache_flag {

        // no hexit to set
        //
        if prep_vector[i+j].ovf_cache_flag { continue }

        // span is 0 hexit
        //
        if prep_vector[i+j].span_flag { continue }

        hexit := 0xf
        if prep_vector[i+j].loq_flag { hexit = 0xe }


        // generic overflow
        //
        if prep_vector[i+j].ovf_flag {
          //cur_v |= 0xf << (4*uint(prep_vector[i+j].hexit_pos))
          cur_v |= uint64(uint(hexit) << (4*uint(prep_vector[i+j].hexit_pos)))
          continue
        }

        cur_v |= uint64( (uint(prep_vector[i+j].knot.TileMapPos) & uint(hexit)) << (4*uint(prep_vector[i+j].hexit_pos)) )

      }

    }

    ret_vec = append(ret_vec, cur_v)
    cur_v=0
  }

  return ret_vec

}

//====   _             _       _                               _ _       _
//====  | | ___   __ _(_)_ __ | |_ ___ _ __ _ __ ___   ___  __| (_) __ _| |_ ___
//====  | |/ _ \ / _` | | '_ \| __/ _ \ '__| '_ ` _ \ / _ \/ _` | |/ _` | __/ _ \
//====  | | (_) | (_| | | | | | ||  __/ |  | | | | | |  __/ (_| | | (_| | ||  __/
//====  |_|\___/ \__, |_|_| |_|\__\___|_|  |_| |_| |_|\___|\__,_|_|\__,_|\__\___|
//====              |_|

func LoqIntermediateCmp(loqi0, loqi1 LoqIntermediate) error {
  if loqi0.count != loqi1.count {
    return fmt.Errorf( fmt.Sprintf("count mismatch: %d != %d", loqi0.count, loqi1.count) )
  }

  if loqi0.code != loqi1.code {
    return fmt.Errorf( fmt.Sprintf("code mismatch: %d != %d", loqi0.code, loqi1.code) )
  }

  if loqi0.stride != loqi1.stride {
    return fmt.Errorf( fmt.Sprintf("stride mismatch: %d != %d", loqi0.stride, loqi1.stride) )
  }

  if len(loqi0.tilepos) != len(loqi1.tilepos) {
    return fmt.Errorf( fmt.Sprintf("tilepos length mismatch: %d != %d", len(loqi0.tilepos), len(loqi1.tilepos)) )
  }

  if len(loqi0.homflag) != len(loqi1.homflag) {
    return fmt.Errorf( fmt.Sprintf("homflag length mismatch: %d != %d", len(loqi0.homflag), len(loqi1.homflag)) )
  }

  for i:=0; i<len(loqi0.homflag); i++ {
    if loqi0.homflag[i] != loqi1.homflag[i] {
      return fmt.Errorf( fmt.Sprintf("homflag mismatch at %d: %d != %d", i, loqi0.homflag[i], loqi1.homflag[i]) )
    }
  }

  if len(loqi0.loqinfo_ints) != len(loqi1.loqinfo_ints) {
    return fmt.Errorf( fmt.Sprintf("loqinfo_ints length mismatch: %d != %d", len(loqi0.loqinfo_ints), len(loqi1.loqinfo_ints)) )
  }

  for i:=0; i<len(loqi0.loqinfo_ints); i++ {
    if loqi0.loqinfo_ints[i] != loqi1.loqinfo_ints[i] {
      return fmt.Errorf( fmt.Sprintf("loqinfo_ints mismatch at %d: %d != %d", i, loqi0.loqinfo_ints[i], loqi1.loqinfo_ints[i]) )
    }
  }

  // we don't care about trailing overflow from the byte
  //
  if ((len(loqi0.loq_flag)+7)/8) != ((len(loqi1.loq_flag)+7)/8) {
    return fmt.Errorf( fmt.Sprintf("loq_flag length mismatch: %d != %d", len(loqi0.loq_flag), len(loqi1.loq_flag)) )
  }

  mm := len(loqi0.loq_flag)
  if len(loqi1.loq_flag) < mm { mm = len(loqi1.loq_flag) }
  for i:=0; i<mm; i++ {
    if loqi0.loq_flag[i] != loqi1.loq_flag[i] {
      return fmt.Errorf( fmt.Sprintf("loq_flag mismatch at %d: %v != %v", i, loqi0.loq_flag[i], loqi1.loq_flag[i]) )
    }
  }

  return nil

}

func LoqIntermediateFromBytes(b []byte) (LoqIntermediate,int) {
  loqi := LoqIntermediate{}

  var dummy uint64
  var dn int ; _ = dn

  n:=0

  dummy = byte2uint64(b[n:n+8])
  n+=8

  rec_count := int(dummy)
  loqi.count = rec_count

  //DEBUG
  //fmt.Printf("# rec_count %v\n", rec_count)

  dummy = byte2uint64(b[n:n+8])
  n+=8

  code := int(dummy)
  loqi.code = code

  //DEBUG
  //fmt.Printf("# code %v\n", code)

  dummy = byte2uint64(b[n:n+8])
  n+=8

  stride := int(dummy)
  loqi.stride = stride

  //DEBUG
  //fmt.Printf("# stride %v\n", stride)

  offset_idx := make([]int, (rec_count+stride-1)/stride)
  for i:=0; i<(rec_count+stride-1)/stride; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8

    offset_idx[i] = int(dummy)
  }


  tilepos_idx := make([]int, (rec_count+stride-1)/stride)
  for i:=0; i<(rec_count+stride-1)/stride; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8

    tilepos_idx[i] = int(dummy)
  }


  homflag := make([]byte, (rec_count+7)/8)
  for i:=0; i<(rec_count+7)/8; i++ {
    homflag[i] = b[n]
    n++
  }

  for i:=0; i<rec_count; i++ {
    tf := false
    if (homflag[i/8] & (1<<uint(i%8)))!=0 { tf = true }
    loqi.homflag = append(loqi.homflag, tf)
  }

  cur_tilepos := 0
  for i:=0; i<rec_count; i++ {
    if (i%loqi.stride) == 0 {
      cur_tilepos = tilepos_idx[i/loqi.stride]
    }
    loqi.tilepos = append(loqi.tilepos, cur_tilepos)
  }



  // loq flag size on vector
  //
  dummy = byte2uint64(b[n:n+8])
  n+=8

  loqflag_bytecount := int(dummy)

  //DEBUG
  //fmt.Printf("# loqflag_bytecount %v\n", loqflag_bytecount)

  loq_flag_vec := b[n:n+loqflag_bytecount]
  n+=loqflag_bytecount

  for i:=0; i<8*loqflag_bytecount; i++ {
    tf := false
    if (loq_flag_vec[i/8] & (1<<uint(i%8))) != 0 { tf = true }
    loqi.loq_flag = append(loqi.loq_flag, tf)
  }


  // size of loq array
  //
  dummy = byte2uint64(b[n:n+8])
  n+=8

  loq_info_byte_count := int(dummy) ; _ = loq_info_byte_count

  //DEBUG
  //fmt.Printf("# loq_info_byte_count %v\n", loq_info_byte_count)


  // main loq array
  //
  loqi.loqi_info = make(map[int]CGFIntermediate)

  cur_step:=0
  //max_step := loq_info_byte_count*8
  max_step := len(loqi.loq_flag)

  //DEBUG
  //fmt.Printf("# loq max_step %d (0x%x)\n", max_step, max_step)

  rec_pos:=0
  byte_offset := 0
  for byte_offset < loq_info_byte_count {

    //DEBUG
    //fmt.Printf("#>> cp0: byte_offset %d, loq_info_byte_count %d, cur_step %d, max_step %d\n", byte_offset, loq_info_byte_count, cur_step, max_step)

    for (cur_step<max_step) && (!loqi.loq_flag[cur_step]) {
      cur_step++
    }

    //DEBUG
    //fmt.Printf("# len(loq_flag) %d\n", len(loqi.loq_flag))
    //fmt.Printf("#>>> cur_step %d (/%d)\n", cur_step, max_step)

    if cur_step==max_step {
      panic( fmt.Sprintf("ERROR: cur_step (%d) == max_step (%d)\n", cur_step, max_step) )
    }

    cgfi := CGFIntermediate{}
    cgfi.loq_flag = true
    cgfi.step = make([][]int, 2)
    cgfi.seq= make([][]string, 2)
    cgfi.varid = make([][]int, 2)
    cgfi.span = make([][]int, 2)
    cgfi.loq = make([][]bool, 2)
    cgfi.nocall_start_len = make([][][]int, 2)

    ntile := make([]int, 1)

    dummy,dn := dlug.ConvertUint64(b[n:])
    n+=dn
    byte_offset+=dn

    ntile[0] = int(dummy)
    loqi.loqinfo_ints = append(loqi.loqinfo_ints, int(ntile[0]))

    cgfi.nocall_start_len[0] = make([][]int, ntile[0])

    if !loqi.homflag[rec_pos] {

      dummy,dn := dlug.ConvertUint64(b[n:])
      n+=dn
      byte_offset+=dn

      ntile = append(ntile, int(dummy))

      loqi.loqinfo_ints = append(loqi.loqinfo_ints, int(ntile[1]))
      cgfi.nocall_start_len[1] = make([][]int, ntile[1])

    } else {
      cgfi.nocall_start_len[1] = make([][]int, ntile[0])
    }

    for allele:=0; allele<len(ntile); allele++ {

      for i:=0; i<ntile[allele]; i++ {


        dummy,dn := dlug.ConvertUint64(b[n:])
        n+=dn
        byte_offset+=dn

        m := int(dummy)
        loqi.loqinfo_ints = append(loqi.loqinfo_ints, int(m))

        for j:=0; j<m; j+=2 {
          dummy,dn := dlug.ConvertUint64(b[n:])
          n+=dn
          byte_offset+=dn

          delpos:=int(dummy)
          cgfi.nocall_start_len[allele][i] = append(cgfi.nocall_start_len[allele][i], delpos)

          if len(ntile)==1 {
            cgfi.nocall_start_len[1][i] = append(cgfi.nocall_start_len[1][i], delpos)
          }

          dummy,dn = dlug.ConvertUint64(b[n:])
          n+=dn
          byte_offset+=dn

          l:=int(dummy)
          cgfi.nocall_start_len[allele][i] = append(cgfi.nocall_start_len[allele][i], l)

          if len(ntile)==1 {
            cgfi.nocall_start_len[1][i] = append(cgfi.nocall_start_len[1][i], l)
          }

          loqi.loqinfo_ints = append(loqi.loqinfo_ints, delpos)
          loqi.loqinfo_ints = append(loqi.loqinfo_ints, l)
        }
      }
    }

    loqi.loqi_info[cur_step] = cgfi
    rec_pos++
    cur_step++

  }

  return loqi,n
}

func BytesFromLoqIntermediate(loqi LoqIntermediate) []byte {
  buf := make([]byte, 64)
  loq_bytes := make([]byte, 0, 1024)

  tobyte64(buf, uint64(loqi.count))
  loq_bytes = append(loq_bytes, buf[0:8]...)

  tobyte64(buf, uint64(loqi.code))
  loq_bytes = append(loq_bytes, buf[0:8]...)

  tobyte64(buf, uint64(loqi.stride))
  loq_bytes = append(loq_bytes, buf[0:8]...)

  offset_idx := make([]uint64, 0, 1024)
  tilepos_idx := make([]uint64, 0, 1024)

  loqinfo_bytes := make([]byte, 0, 1024)

  loq_flag := make([]byte, (len(loqi.loq_flag) + 7)/8)
  for i:=0; i<len(loqi.loq_flag); i++ {
    if loqi.loq_flag[i] { loq_flag[i/8] |= 1<<uint(i%8) }
  }

  cur_rec := 0
  byte_offset := 0 ; _ = byte_offset
  p:=0
  for p<len(loqi.loqinfo_ints) {

    if (cur_rec%loqi.stride) == 0 {
      offset_idx = append(offset_idx, uint64(len(loqinfo_bytes)))
      tilepos_idx = append(tilepos_idx, uint64(loqi.tilepos[cur_rec]))
    }

    ntile := make([]int, 1)

    ma := dlug.MarshalUint64(uint64(loqi.loqinfo_ints[p]))
    loqinfo_bytes = append(loqinfo_bytes, ma...)

    ntile[0]=loqi.loqinfo_ints[p]
    p++

    if !loqi.homflag[cur_rec] {

      mb := dlug.MarshalUint64(uint64(loqi.loqinfo_ints[p]))
      loqinfo_bytes = append(loqinfo_bytes, mb...)

      ntile = append(ntile, loqi.loqinfo_ints[p])
      p++
    }

    for allele:=0; allele<len(ntile); allele++ {
      for i:=0; i<ntile[allele]; i++ {

        m:=loqi.loqinfo_ints[p]
        p++

        mb := dlug.MarshalUint64(uint64(m))
        loqinfo_bytes = append(loqinfo_bytes, mb...)

        for j:=0; j<m; j+=2 {
          pos := loqi.loqinfo_ints[p]
          p++

          l := loqi.loqinfo_ints[p]
          p++

          x := dlug.MarshalUint64(uint64(pos))
          loqinfo_bytes = append(loqinfo_bytes, x...)

          y := dlug.MarshalUint64(uint64(l))
          loqinfo_bytes = append(loqinfo_bytes, y...)
        }

      }

    }

    cur_rec++

  }

  loq_info_byte_count := len(loqinfo_bytes)

  homflag := make([]byte, (cur_rec+7)/8)
  for i:=0; i<cur_rec; i++ {

    if loqi.homflag[i] {
      homflag[i/8] |= 1<<uint(i%8)
    }
  }

  for i:=0; i<len(offset_idx); i++ {
    tobyte64(buf, offset_idx[i])
    loq_bytes = append(loq_bytes, buf[0:8]...)
  }

  for i:=0; i<len(tilepos_idx); i++ {
    tobyte64(buf, tilepos_idx[i])
    loq_bytes = append(loq_bytes, buf[0:8]...)
  }

  loq_bytes = append(loq_bytes, homflag...)

  tobyte64(buf, uint64(len(loq_flag)))
  loq_bytes = append(loq_bytes, buf[0:8]...)
  loq_bytes = append(loq_bytes, loq_flag...)

  tobyte64(buf, uint64(loq_info_byte_count))
  loq_bytes = append(loq_bytes, buf[0:8]...)
  loq_bytes = append(loq_bytes, loqinfo_bytes...)

  return loq_bytes
}

func _loq_hom(nocall_start_len [][][]int) bool {
  if len(nocall_start_len)==1 { return true }
  if len(nocall_start_len)!=2 { return false }

  a := nocall_start_len[0]
  b := nocall_start_len[1]

  if len(a)!=len(b) { return false }
  for i:=0; i<len(a); i++ {
    if len(a[i]) != len(b[i]) { return false }
    for j:=0; j<len(a[i]); j++ {
      if a[i][j]!=b[i][j] { return false }
    }
  }

  return true
}

func (ctx *CGFContext) ConstructLoqIntermediate(prep_vector []VectorElement) LoqIntermediate {
  loqi := LoqIntermediate{}

  loqi.code = 0
  loqi.stride = 256

  // fill out loq bit vector
  //
  for i:=0; i<len(prep_vector); i++ {

    if prep_vector[i].span_flag {
      loqi.loq_flag = append(loqi.loq_flag, false)
      continue
    }

    if prep_vector[i].loq_flag {
      loqi.loq_flag = append(loqi.loq_flag, true)
    } else {
      loqi.loq_flag = append(loqi.loq_flag, false)
    }

  }

  loqi.count = 0

  // populate loqinfo_ints
  //
  for i:=0; i<len(prep_vector); i++ {
    if prep_vector[i].loq_flag {
      loqi.tilepos = append(loqi.tilepos, i)
      loqi.count++

      nocall_start_len := prep_vector[i].knot.nocall_start_len

      if _loq_hom(nocall_start_len) {

        // Hom
        //

        loqi.homflag = append(loqi.homflag, true)
        loqi.loqinfo_ints = append(loqi.loqinfo_ints, len(nocall_start_len[0]))

        for ii:=0; ii<len(nocall_start_len[0]); ii++ {
          loqi.loqinfo_ints = append(loqi.loqinfo_ints, len(nocall_start_len[0][ii]))

          start := 0
          for jj:=0; jj<len(nocall_start_len[0][ii]); jj+=2 {
            loqi.loqinfo_ints = append(loqi.loqinfo_ints, nocall_start_len[0][ii][jj]-start)
            loqi.loqinfo_ints = append(loqi.loqinfo_ints, nocall_start_len[0][ii][jj+1])
            start = nocall_start_len[0][ii][jj]
          }

        }


      } else {

        // Het
        //
        loqi.homflag = append(loqi.homflag, false)
        loqi.loqinfo_ints = append(loqi.loqinfo_ints, len(nocall_start_len[0]))
        loqi.loqinfo_ints = append(loqi.loqinfo_ints, len(nocall_start_len[1]))

        for allele:=0; allele<2; allele++ {
          for ii:=0; ii<len(nocall_start_len[allele]); ii++ {
            loqi.loqinfo_ints = append(loqi.loqinfo_ints, len(nocall_start_len[allele][ii]))

            start := 0
            for jj:=0; jj<len(nocall_start_len[allele][ii]); jj+=2 {
              loqi.loqinfo_ints = append(loqi.loqinfo_ints, nocall_start_len[allele][ii][jj]-start)
              loqi.loqinfo_ints = append(loqi.loqinfo_ints, nocall_start_len[allele][ii][jj+1])
              start = nocall_start_len[allele][ii][jj]
            }

          }
        }

      }

    }
  }


  return loqi
}

//====               _   _     _       _                               _ _       _
//====   _ __   __ _| |_| |__ (_)_ __ | |_ ___ _ __ _ __ ___   ___  __| (_) __ _| |_ ___
//====  | '_ \ / _` | __| '_ \| | '_ \| __/ _ \ '__| '_ ` _ \ / _ \/ _` | |/ _` | __/ _ \
//====  | |_) | (_| | |_| | | | | | | | ||  __/ |  | | | | | |  __/ (_| | | (_| | ||  __/
//====  | .__/ \__,_|\__|_| |_|_|_| |_|\__\___|_|  |_| |_| |_|\___|\__,_|_|\__,_|\__\___|
//====  |_|

func BytesFromPathIntermediate(pathi PathIntermediate) []byte {

  PathBytes := make([]byte, 0, 1024)

  buf := make([]byte, 64)

  ns := len(pathi.name)
  ns_bytes := dlug.MarshalUint64(uint64(ns))
  PathBytes = append(PathBytes, ns_bytes...)
  PathBytes = append(PathBytes, []byte(pathi.name)...)

  tobyte64(buf, uint64(pathi.ntile))
  PathBytes = append(PathBytes, buf[0:8]...)
  for i:=0; i<len(pathi.VecUint64); i++ {
    tobyte64(buf, uint64(pathi.VecUint64[i]))
    PathBytes = append(PathBytes, buf[0:8]...)
  }

  ovf_bytes := BytesFromOverflowIntermediate(pathi.ofsi)
  fovf_bytes := BytesFromFinalOverflowIntermediate(pathi.fofsi)
  loq_bytes := BytesFromLoqIntermediate(pathi.loqi)

  PathBytes = append(PathBytes, ovf_bytes...)
  PathBytes = append(PathBytes, fovf_bytes...)
  PathBytes = append(PathBytes, loq_bytes...)


  return PathBytes
}

func PathIntermediateFromBytes(b []byte) (PathIntermediate,int) {
  pathi := PathIntermediate{}
  n:=0

  dummy,dn := dlug.ConvertUint64(b[n:])
  n+=dn

  ns := int(dummy)

  pathi.name = string(b[n:n+ns])
  n+=ns

  dummy = byte2uint64(b[n:n+8])
  n+=8

  ntile := int(dummy)
  veclen := (ntile+31)/32

  pathi.ntile = ntile

  for i:=0; i<veclen; i++ {
    dummy = byte2uint64(b[n:n+8])
    n+=8
    pathi.VecUint64 = append(pathi.VecUint64, dummy)
  }

  pathi.ofsi,dn = OverflowIntermediateFromBytes(b[n:])
  n+=dn

  pathi.fofsi,dn = FinalOverflowIntermediateFromBytes(b[n:])
  n+=dn

  pathi.loqi,dn = LoqIntermediateFromBytes(b[n:])
  n+=dn

  return pathi,n
}

func _loq_skip(loqi LoqIntermediate, step int) int {
  pos:=0
  for i:=0; i<len(loqi.tilepos); i++ {
    anchor_step := loqi.tilepos[i]
    if anchor_step == step { return pos }

    ntile := make([]int, 1)
    ntile[0] = loqi.loqinfo_ints[pos]
    pos++

    if loqi.homflag[anchor_step] {
      ntile = append(ntile, loqi.loqinfo_ints[pos])
      pos++
    }

    for allele:=0; allele<len(ntile); allele++ {
      for ii:=0; ii<ntile[allele]; ii++ {
        l := loqi.loqinfo_ints[pos]
        pos++

        for jj:=0; jj<l; jj+=2 {
          delpos := loqi.loqinfo_ints[pos] ; _ = delpos
          pos++

          loqlen := loqi.loqinfo_ints[pos] ; _ = loqlen
          pos++
        }
      }
    }

  }

  return pos
}


func construct_loq_map(loqi LoqIntermediate) map[int]map[int][]int {
  loq_map := make(map[int]map[int][]int)

  pos := 0
  for i:=0; i<len(loqi.tilepos); i++ {
    anchor_step := loqi.tilepos[i]

    //fmt.Printf("construct_loq_map anchor_step %d\n", anchor_step)

    loq_map[anchor_step] = make(map[int][]int)

    ntile := make([]int, 1)
    ntile[0] = loqi.loqinfo_ints[pos]
    pos++

    //fmt.Printf("  %v\n", ntile)

    if loqi.homflag[anchor_step] {
      ntile = append(ntile, loqi.loqinfo_ints[pos])
      pos++

      //fmt.Printf(" +%v\n", ntile)

    }

    for allele:=0; allele<len(ntile); allele++ {
      for ii:=0; ii<ntile[allele]; ii++ {
        l := loqi.loqinfo_ints[pos]
        pos++

        //fmt.Printf("   [%d][%d] m %d\n", allele, ii, l)


        for jj:=0; jj<l; jj+=2 {
          delpos := loqi.loqinfo_ints[pos] ; _ = delpos
          pos++

          //fmt.Printf("      [%d][%d] delpos %d\n", allele, ii, delpos)

          loqlen := loqi.loqinfo_ints[pos] ; _ = loqlen
          pos++

          //fmt.Printf("      [%d][%d] loqlen %d\n", allele, ii, loqlen)


        }
      }
    }

  }

  return loq_map

}


//====                  _ _
//====    ___ _ __ ___ (_) |_
//====   / _ \ '_ ` _ \| | __|
//====  |  __/ | | | | | | |_
//====   \___|_| |_| |_|_|\__|


//func emit_intermediate(ctx *CGFContext, path_idx int, allele_path [][]TileInfo) error {
//func emit_PathBytes(ctx *CGFContext, path_idx int, allele_path [][]TileInfo) ([]byte, error) {
func (ctx *CGFContext) EmitPathBytes(path_idx int, allele_path [][]TileInfo) ([]byte, error) {
  debug_output:=false
  //debug_output:=true

  max_tile := 0

  cgf := ctx.CGF ; _ = cgf
  sglf := ctx.SGLF

  span_sum := 0
  step_idx0,step_idx1 := 0,0

  knot := CGFIntermediate{}
  _init_knot(&knot)

  tileKnot := make([]CGFIntermediate, 0, 1024)

  /*
  //DEBUG
  fmt.Printf(">>> len (%d, %d) len sglf.LibInfo (%d)\n", len(allele_path[0]), len(allele_path[1]), len(sglf.LibInfo))
  for k := range sglf.LibInfo {
    fmt.Printf("  len sglf.LibInfo (%d)\n", len(sglf.LibInfo[k]))
  }
  */


  // Construct the intermediate string of knots
  //
  for (step_idx0<len(allele_path[0])) || (step_idx1<len(allele_path[1])) {
  //for (step_idx0<len(allele_path[0])) && (step_idx1<len(allele_path[1])) {

  /*
    //DEBUG
    fmt.Printf(">> step_idx0 %d (step %d %x (+%d)) step_idx1 %d (step %d %x (+%d))\n",
      step_idx0,
      allele_path[0][step_idx0].Step, allele_path[0][step_idx0].Step, allele_path[0][step_idx0].Span,
      step_idx1,
      allele_path[1][step_idx1].Step, allele_path[1][step_idx1].Step, allele_path[1][step_idx1].Span)
    fmt.Printf("  span_sum %d (before)\n", span_sum)
    */

    if span_sum >= 0 {

      //DEBUG
      //fmt.Printf(">> step_idx0 %d (%d), step_idx1 %d (%d)\n", step_idx0, len(allele_path[0]), step_idx1, len(allele_path[1]))

      span_count,e := _add_knot(&knot, 0, step_idx0, allele_path[0][step_idx0], sglf)
      if e!=nil {
        panic(e)
      }

      step_idx0++
      span_sum -= span_count
    } else {
      span_count,e := _add_knot(&knot, 1, step_idx1, allele_path[1][step_idx1], sglf)
      if e!=nil {
        panic(e)
      }

      step_idx1++
      span_sum += span_count
    }

    /*
    //DEBUG
    fmt.Printf("  (after) step_idx0 %d (step %d %x (+%d)) step_idx1 %d (step %d %x (+%d))\n",
      step_idx0,
      allele_path[0][step_idx0].Step, allele_path[0][step_idx0].Step, allele_path[0][step_idx0].Span,
      step_idx1,
      allele_path[1][step_idx1].Step, allele_path[1][step_idx1].Step, allele_path[1][step_idx1].Span)
    fmt.Printf("  span_sum %d (after)\n", span_sum)
    */

    if span_sum==0 {

      _knot_tot_span(&knot)
      knot.TileMapKey = create_tilemap_string_lookup2(knot.varid[0], knot.span[0], knot.varid[1], knot.span[1])
      tileKnot = append(tileKnot, knot)

      if max_tile < (knot.step[0][0] + knot.tot_span) {
        max_tile = knot.step[0][0] + knot.tot_span
      }

      knot = CGFIntermediate{}
      _init_knot(&knot)
    }

  }

  // Sanity check
  if ( ((len(allele_path[0])-step_idx0) < 0) ||
       ((len(allele_path[0])-step_idx0) > 1) ||
       ((len(allele_path[1])-step_idx1) < 0) ||
       ((len(allele_path[1])-step_idx1) > 1) ) {
  //if (step_idx0 != len(allele_path[0])) || (step_idx1 != len(allele_path[1])) {
    panic(fmt.Sprintf("step indexes do not match: step_idx0 %d != %d or step_idx1 %d != %d\n",
      step_idx0, len(allele_path[0]), step_idx1, len(allele_path[1])))
  }

  // Prep for binary representation
  //
  prep_vector := make([]VectorElement, 0, 1024)
  cache_counter := 0

  for ind:=0; ind<len(tileKnot); ind++ {
    knot := tileKnot[ind]

    cur_step := knot.step[0][0]
    next_step := cur_step + knot.tot_span

    if (cur_step%(32))==0 {
      cache_counter = 0
    }

    vec_ele := VectorElement{}

    // We have a canonical tile.  Add it and move on
    //
    if (!knot.loq_flag) && (knot.tot_span == 1) && (knot.varid[0][0] == 0) && (knot.varid[1][0]==0) {
      vec_ele.canon_flag = true
      vec_ele.knot = knot
      prep_vector = append(prep_vector, vec_ele)
      continue
    }

    if cache_counter > (32/4) {
      vec_ele.ovf_cache_flag = true
    }
    cache_counter++

    if knot.loq_flag { vec_ele.loq_flag = true }

    if _,ok := ctx.TileMapLookup[knot.TileMapKey] ; ok {
      // We've found it in the TileMap.  We can either
      // put it into the vector cache or we can put it into
      // the overflow table.  If it's either low quality
      // or more than 0xd, it goes into the overflow table.
      // Otherwise it can go in the vector cache.
      //

      //knot.tilemap_pos = tilemap_pos.TileMap
      knot.TileMapPos = ctx.TileMapPosition[knot.TileMapKey]

      if knot.TileMapPos >= 0xd { vec_ele.ovf_flag = true }
      if vec_ele.loq_flag { vec_ele.ovf_flag = true }

      if cache_counter > (32/4) {
        vec_ele.ovf_cache_flag = true
        vec_ele.ovf_flag = true
      } else {
        vec_ele.cache_flag = true
        vec_ele.hexit_pos = cache_counter-1
      }



      // If our cache can still hold hexits and our tilemap
      // entry isn't too big and it's high quality.
      //
      //if !vec_ele.ovf_cache_flag && !vec_ele.ovf_flag {
      //  vec_ele.cache_flag = true
      //  vec_ele.hexit_pos = cache_counter
      //}

    } else {

      // We couldn't find it in the TileMap table, so
      // we store it in the FinalOverflowMap table
      //

      vec_ele.fin_ovf_flag = true
      vec_ele.ovf_flag = true
      //if !vec_ele.ovf_cache_flag && !vec_ele.ovf_flag {
      if !vec_ele.ovf_cache_flag {

        if cache_counter > (32/4) {
          vec_ele.ovf_cache_flag = true
        } else {
          vec_ele.cache_flag = true
          vec_ele.hexit_pos = cache_counter-1
        }
      }

    }

    vec_ele.knot = knot
    prep_vector = append(prep_vector, vec_ele)

    // Add an explicit entry for spanning tiles
    //
    cur_step++
    for ; cur_step<next_step; cur_step++ {
      if (cur_step%(32))==0 {
        cache_counter = 0
      }
      cache_counter++

      span_vec_ele := VectorElement{}
      span_vec_ele.canon_flag = false
      span_vec_ele.span_flag = true

      if cache_counter > (32/4) {
        span_vec_ele.ovf_cache_flag = true
        span_vec_ele.ovf_flag = true
      } else {
        span_vec_ele.cache_flag = true
        span_vec_ele.hexit_pos = cache_counter-1
      }

      prep_vector = append(prep_vector, span_vec_ele)
    }

  }

  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================
  // ======================================================

  if debug_output {
    for i:=0; i<len(prep_vector); i++ {

      if (i%32) == 0 {
        fmt.Printf("#    con(^) ca(c) oca(>) ovf(/) fin(!) sp(~) lq(*) hexit vec ...\n")
      }

      if prep_vector[i].span_flag {
        //fmt.Printf("[%d+_] cache %v, ovf_cache %v\n", i, prep_vector[i].cache_flag, prep_vector[i].ovf_cache_flag)
        fmt.Printf("[%4x+_] ", i)
        //fmt.Printf(" con %v ca %v oca %v ovf %v fin %v sp %v lq %v x %v v %v ",
        fmt.Printf(" %v %v %v %v %v %v %v %v %v tmap(%d) ",
          _tf_(prep_vector[i].canon_flag, "^"),
          _tf_(prep_vector[i].cache_flag, "c"),
          _tf_(prep_vector[i].ovf_cache_flag, ">"),
          _tf_(prep_vector[i].ovf_flag, "/"),
          _tf_(prep_vector[i].fin_ovf_flag, "!"),
          _tf_(prep_vector[i].span_flag, "~"),
          _tf_(prep_vector[i].loq_flag, "*"),
          prep_vector[i].hexit_pos,
          prep_vector[i].vec_pos,
          prep_vector[i].knot.TileMapPos)

        fmt.Printf("\n")

      } else {
        knot := prep_vector[i].knot
        fmt.Printf("[%4x+%x] ", knot.step[0][0], knot.tot_span)

        if prep_vector[i].cache_flag && (prep_vector[i].hexit_pos==0) {
          fmt.Printf(" %v %v %v %v %v %v %v %v %v tmap(%d) ",
            _tf_(prep_vector[i].canon_flag, "^"),
            _tf_(prep_vector[i].cache_flag, "c"),
            _tf_(prep_vector[i].ovf_cache_flag, ">"),
            _tf_(prep_vector[i].ovf_flag, "/"),
            _tf_(prep_vector[i].fin_ovf_flag, "!"),
            _tf_(prep_vector[i].span_flag, "~"),
            _tf_(prep_vector[i].loq_flag, "*"),
            "$",
            prep_vector[i].vec_pos,
            prep_vector[i].knot.TileMapPos)

        } else {
          fmt.Printf(" %v %v %v %v %v %v %v %v %v tmap(%d) ",
            _tf_(prep_vector[i].canon_flag, "^"),
            _tf_(prep_vector[i].cache_flag, "c"),
            _tf_(prep_vector[i].ovf_cache_flag, ">"),
            _tf_(prep_vector[i].ovf_flag, "/"),
            _tf_(prep_vector[i].fin_ovf_flag, "!"),
            _tf_(prep_vector[i].span_flag, "~"),
            _tf_(prep_vector[i].loq_flag, "*"),
            prep_vector[i].hexit_pos,
            prep_vector[i].vec_pos,
            prep_vector[i].knot.TileMapPos)
        }

        for allele:=0; allele<2; allele++ {
          if allele==0 {
            fmt.Printf("A(");
          } else {
            fmt.Printf(") : B(");
          }
          for j:=0; j<len(knot.varid[allele]); j++ {
            if j>0 { fmt.Printf(":") }
            ch := "_"
            if knot.loq[allele][j] { ch = "*" }
            fmt.Printf("%x%s%x+%x",
              knot.step[allele][j],
              ch,
              knot.varid[allele][j],
              knot.span[allele][j])
          }
        }
        fmt.Printf(")")

        fmt.Printf("\n")

      }

    }
  }

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  vec64 := ctx.ConstructUint64Vector(prep_vector)

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  if debug_output {
    vec_bytes := make([]byte, 8*len(vec64))
    for i:=0; i<len(vec64); i++ {
      tobyte64(vec_bytes[8*i:], vec64[i])
    }

    fmt.Printf(">>>returned vec %d\n", len(vec64))
    for i:=0; i<len(vec64); i++ {
      if (i%4)==0 { fmt.Printf("\n") }
      fmt.Printf("[%3x,%4x] %8x.%8x |", i, 32*i, (vec64[i]&(0xffffffff<<32))>>32, vec64[i]&0xffffffff)
    }
    fmt.Printf("\n")


    random_start := 0x12af
    random_n := 0x120
    random_ovf_count := 0
    for i:=random_start; i<(random_start+random_n); i++ {
      if prep_vector[i].ovf_flag || prep_vector[i].ovf_cache_flag {
        random_ovf_count++
      }
    }

    check_ovf_count := CountOverflowVectorUint64(vec64, random_start, random_start+random_n)

    fmt.Printf("CHECKING (step %x+%x(%x)) real:%d check:%d\n", random_start, random_n, random_start+random_n, random_ovf_count, check_ovf_count)


    if debug_output {

      for i:=0; i<len(tileKnot); i++ {
        fmt.Printf("[%4x+%x] ", tileKnot[i].step[0][0], tileKnot[i].tot_span)

        for allele:=0; allele<2; allele++ {
          if allele==0 {
            fmt.Printf("A(");
          } else {
            fmt.Printf(") : B(");
          }
          for j:=0; j<len(tileKnot[i].varid[allele]); j++ {
            if j>0 { fmt.Printf(":") }
            ch := "_"
            if tileKnot[i].loq[allele][j] { ch = "*" }
            fmt.Printf("%x%s%x+%x",
              tileKnot[i].step[allele][j],
              ch,
              tileKnot[i].varid[allele][j],
              tileKnot[i].span[allele][j])
          }
        }
        fmt.Printf(")\n")

      }


      fmt.Printf("LOQ INFO (tileKnot)\n")
      for i:=0; i<len(tileKnot); i++ {
        fmt.Printf("[%4x+%x] loq ", tileKnot[i].step[0][0], tileKnot[i].tot_span)

        for allele:=0; allele<2; allele++ {
          if allele==0 {
            fmt.Printf("A(");
          } else {
            fmt.Printf(") : B(");
          }
          for j:=0; j<len(tileKnot[i].nocall_start_len[allele]); j++ {
            if j>0 { fmt.Printf(",") }

            fmt.Printf("{%d}", j)
            for k:=0; k<len(tileKnot[i].nocall_start_len[allele][j]); k+=2 {
              fmt.Printf(";%d+%d",
                tileKnot[i].nocall_start_len[allele][j][k],
                tileKnot[i].nocall_start_len[allele][j][k+1])
            }
          }
        }
        fmt.Printf(")\n")

      }

    }


  }


  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  ofsi := ctx.ConstructOffsetIntermediate(prep_vector)

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================


  if debug_output {

    fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>\n")

    fmt.Printf("BYTES FROM ofsi to byte_ofsi_test0\n")
    byte_ofsi_test0 := BytesFromOverflowIntermediate(ofsi) ; _ = byte_ofsi_test0


    fmt.Printf("BYTES TO byte_ofsi_test0 to ofsi_throwaway\n")
    ofsi_throwaway,dn0 := OverflowIntermediateFromBytes(byte_ofsi_test0) ; _ = ofsi_throwaway

    for i:=0; i<len(ofsi_throwaway.tilepos_idx); i++ {
      fmt.Printf("  hrm: ofsi_throwaway.tilepos_idx[%d] %d\n", i, ofsi_throwaway.tilepos_idx[i])
    }

    err := OverflowIntermediateCmp(ofsi_throwaway, ofsi)
    fmt.Printf(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>> %v\n", err)

    fmt.Printf("BYTES FROM ofsi_throwaway to byte_ofsi_test1\n")
    byte_ofsi_test1 := BytesFromOverflowIntermediate(ofsi_throwaway)
    _ = byte_ofsi_test1


    fmt.Printf("BYTES TO byte_ofsi_test1 to ofsi_throwaway1\n")
    ofsi_throwaway1,dn1 := OverflowIntermediateFromBytes(byte_ofsi_test0) ; _ = ofsi_throwaway1

    fmt.Printf(" dn0 %d, dn1 %d\n", dn0, dn1)

    fmt.Printf("!!! %d %d\n", len(byte_ofsi_test0), len(byte_ofsi_test1))
    for i:=0; i<len(byte_ofsi_test0); i++ {
      if byte_ofsi_test0[i] != byte_ofsi_test1[i] {
        fmt.Printf("mismatched byte at %d (%d != %d)\n", i, byte_ofsi_test0[i], byte_ofsi_test1[i])
      }
    }

    fmt.Printf("OFSI (%d (%x))\n", len(ofsi.TileMap), len(ofsi.TileMap))
    for i:=0; i<len(ofsi.TileMap); i++ {
      fmt.Printf("[%d] {%x} %d (%x) fovf?%v span?%v\n", i, ofsi.tilepos[i], ofsi.TileMap[i], ofsi.TileMap[i], ofsi.final_overflow_flag[i], ofsi.span_flag[i])
    }

    fmt.Printf("OFSI CHECK\n")
    stride_tile_pos := 0
    stride_ofs := 0
    for i:=0; i<len(ofsi.TileMap); i++ {

      if i%256 == 0 {
        stride_tile_pos = ofsi.tilepos[i]
        stride_ofs = i
      }

      check_ovf_count := CountOverflowVectorUint64(vec64, stride_tile_pos, ofsi.tilepos[i])
      check_ovf_count += stride_ofs

      fmt.Printf("[%d] {%x} %d (%x) ==> ovf_count: %d (%x)\n", i, ofsi.tilepos[i], ofsi.TileMap[i], ofsi.TileMap[i], check_ovf_count, check_ovf_count)
      if check_ovf_count != i {
        real_ovf_count := VectorElementOverflowCount(prep_vector, stride_tile_pos, ofsi.tilepos[i] - stride_tile_pos)
        fmt.Printf("ERROR!!!! check_ovf_count %d != i %d (real %d)\n", check_ovf_count, i, real_ovf_count)


      }


    }

  }


  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  fofsi := ctx.ConstructFinalOffsetIntermediate(prep_vector)

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================


  if debug_output {

    fmt.Printf("FOFSI (%d (%x))\n", len(fofsi.tilepos), len(fofsi.tilepos))

    cur_int_pos := 0

    for i:=0; i<len(fofsi.tilepos); i++ {
      fmt.Printf("[%d] tilepos{%x}", i, fofsi.tilepos[i])

      step := fofsi.variant_ints[cur_int_pos] ; cur_int_pos++
      nallele := fofsi.variant_ints[cur_int_pos] ; cur_int_pos++

      fmt.Printf(" step:%x N(%d)", step, nallele)

      n_a_allele := fofsi.variant_ints[cur_int_pos] ; cur_int_pos++
      fmt.Printf(" A(%d)[", n_a_allele)

      for ii:=0; ii<n_a_allele; ii++ {
        fmt.Printf(" %x+%x", fofsi.variant_ints[cur_int_pos], fofsi.variant_ints[cur_int_pos+1])
        cur_int_pos+=2
      }
      fmt.Printf(" ]")

      n_b_allele := fofsi.variant_ints[cur_int_pos] ; cur_int_pos++
      fmt.Printf(" B(%d)[", n_b_allele)

      for ii:=0; ii<n_b_allele; ii++ {
        fmt.Printf(" %x+%x", fofsi.variant_ints[cur_int_pos], fofsi.variant_ints[cur_int_pos+1])
        cur_int_pos+=2
      }
      fmt.Printf(" ]\n")


    }

    fofsi_bytes0 := BytesFromFinalOverflowIntermediate(fofsi) ; _ = fofsi_bytes0
    fofsi_temp1,dn0 := FinalOverflowIntermediateFromBytes(fofsi_bytes0) ; _ = fofsi_temp1
    fofsi_bytes1 := BytesFromFinalOverflowIntermediate(fofsi_temp1)

    fofsi_cnv,dn1 := FinalOverflowIntermediateFromBytes(fofsi_bytes1)
    fmt.Printf("FOFSI BYTES %d %d (dn %d %d)\n", len(fofsi_bytes0), len(fofsi_bytes1), dn0, dn1)

    if len(fofsi_bytes0) != len(fofsi_bytes1) {
      fmt.Printf("ERROR: length mismatch for fofsi_bytes %d != %d\n", len(fofsi_bytes0), len(fofsi_bytes1))
    } else {
      for i:=0; i<len(fofsi_bytes0); i++ {
        if fofsi_bytes0[i] != fofsi_bytes1[i] {
          fmt.Printf("ERROR: byte mismatch for fofsi_bytes %d: %d != %d\n", i, fofsi_bytes0[i], fofsi_bytes1[i])
        }
      }
    }

    err := FinalOverflowIntermediateCmp(fofsi, fofsi_cnv)
    if err!=nil { fmt.Printf("ERROR: %v\n", err) }

  }

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  loqi := ctx.ConstructLoqIntermediate(prep_vector)
  _ = loqi

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  if debug_output {

    fmt.Printf("LOQ DEBUG\n")
    fmt.Printf("loqi count %d\n", loqi.count)
    fmt.Printf("loqi stride %d\n", loqi.stride)

    p:=0
    for i:=0; i<len(loqi.tilepos); i++ {
      fmt.Printf("{x%x} %v ", loqi.tilepos[i], loqi.homflag[i])

      if loqi.homflag[i] {

        n := loqi.loqinfo_ints[p]
        p++

        fmt.Printf("[%d]", n)

        for ii:=0; ii<n; ii++ {
          m := loqi.loqinfo_ints[p]
          p++

          fmt.Printf(" (%d)", m)

          st:=0
          for jj:=0; jj<m; jj+=2 {
            fmt.Printf(";%d+%d", loqi.loqinfo_ints[p]+st, loqi.loqinfo_ints[p+1])
            st += loqi.loqinfo_ints[p]
            p+=2
          }
        }

      } else {

        n0 := loqi.loqinfo_ints[p]
        p++
        n1 := loqi.loqinfo_ints[p]
        p++

        fmt.Printf(" (%d,%d)", n0,n1)

        for ii:=0; ii<n0; ii++ {
          m := loqi.loqinfo_ints[p]
          p++

          fmt.Printf(" (%d)", m)

          st:=0
          for jj:=0; jj<m; jj+=2 {
            fmt.Printf(";%d+%d", loqi.loqinfo_ints[p]+st, loqi.loqinfo_ints[p+1])
            st += loqi.loqinfo_ints[p]
            p+=2
          }
        }

        fmt.Printf(" :: ")

        for ii:=0; ii<n1; ii++ {
          m := loqi.loqinfo_ints[p]
          p++

          fmt.Printf(" (%d)", m)

          st:=0
          for jj:=0; jj<m; jj+=2 {
            fmt.Printf(";%d+%d", loqi.loqinfo_ints[p]+st, loqi.loqinfo_ints[p+1])
            st += loqi.loqinfo_ints[p]
            p+=2
          }
        }


      }

      fmt.Printf("\n")

    }

    loq_bytes0 := BytesFromLoqIntermediate(loqi) ; _ = loq_bytes0
    loqi_test0,dn := LoqIntermediateFromBytes(loq_bytes0)
    loq_bytes1 := BytesFromLoqIntermediate(loqi_test0)

    fmt.Printf(">>> lOQ CMP: %v\n", LoqIntermediateCmp(loqi, loqi_test0))
    fmt.Printf(">>>>>>>>>>>>>> LOQ FROM/TO BYTES %d %d (dn %d)\n", len(loq_bytes0), len(loq_bytes1), dn)

    if len(loq_bytes0) != len(loq_bytes1) {
      fmt.Printf("ERROR: len(loq_bytes0) %d != len(loq_bytes1) %d\n", len(loq_bytes0), len(loq_bytes1))
    } else {
      for i:=0; i<len(loq_bytes0); i++ {
        if loq_bytes0[i] != loq_bytes1[i] {
          fmt.Printf("byte mismatch at %d: %d != %d\n", i, loq_bytes0[i], loq_bytes1[i])
        }
      }
    }


  }

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  pathi := PathIntermediate{}
  pathi.name = fmt.Sprintf("%04x", path_idx)
  pathi.ntile = max_tile
  pathi.VecUint64 = vec64
  pathi.ofsi = ofsi
  pathi.fofsi = fofsi
  pathi.loqi = loqi

  PathBytes := BytesFromPathIntermediate(pathi)

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  if debug_output {

    pathi_test0,dn := PathIntermediateFromBytes(PathBytes)
    PathBytes1 := BytesFromPathIntermediate(pathi_test0)

    fmt.Printf("pathi::: (ntile %d, %d)\n", pathi.ntile, pathi_test0.ntile)
    fmt.Printf(">>>> len(PathBytes) %d, len(PathBytes1) %d (dn %d)\n", len(PathBytes), len(PathBytes1), dn)

    if len(PathBytes) != len(PathBytes1) {
      fmt.Printf("path byte length mismatch: %d != %d\n", len(PathBytes), len(PathBytes1))
    } else {
      for i:=0; i<len(PathBytes); i++ {
        if PathBytes[i] != PathBytes1[i] {
          fmt.Printf("path byte mismatch at %d: %d != %d\n", i, PathBytes[i], PathBytes1[i])
        }
      }
    }

  }


  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  patho,_ := PathIntermediateFromBytes(PathBytes)

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================

  // FILL IN LOQINTERMEDIATE
  tilemap := UnpackTileMap(ctx.CGF.TileMap)
  for anchor_step := range patho.loqi.loqi_info {
    cgfi := patho.loqi.loqi_info[anchor_step]
    knot := GetKnot(tilemap, patho, anchor_step)

    if knot == nil {
      panic( fmt.Sprintf("anchor_step %d has no knot", anchor_step) )
    }

    for allele:=0; allele<2; allele++ {
      for idx:=0; idx<len(knot[allele]); idx++ {
        cgfi.varid[allele] = append(cgfi.varid[allele], knot[allele][idx].VarId)
        cgfi.span[allele]  = append(cgfi.span[allele], knot[allele][idx].Span)
        cgfi.step[allele]  = append(cgfi.step[allele], knot[allele][idx].Step)
      }
    }

    patho.loqi.loqi_info[anchor_step] = cgfi

  }

  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================
  //=====================================================


  if debug_output {

    fmt.Printf("LOQDUMP\n")
    for i:=0; i<len(patho.loqi.loqinfo_ints); i++ {
      fmt.Printf("[%d] %d\n", i, patho.loqi.loqinfo_ints[i])
    }

    // FILL IN LOQINTERMEDIATE
    /*
    tilemap := unpack_tilemap(ctx.CGF.TileMap)
    for anchor_step := range patho.loqi.loqi_info {
      cgfi := patho.loqi.loqi_info[anchor_step]
      knot := GetKnot(ctx, tilemap, patho, anchor_step)

      if knot == nil {
        panic( fmt.Sprintf("anchor_step %d has no knot", anchor_step) )
      }

      fmt.Printf("???? len knot %d\n", len(knot))

      for allele:=0; allele<2; allele++ {
        for idx:=0; idx<len(knot[allele]); idx++ {
          //cgfi.varid[allele][idx] = knot[allele][idx].VarId
          //cgfi.span[allele][idx] = knot[allele][idx].Span
          //cgfi.step[allele][idx] = knot[allele][idx].Step
          cgfi.varid[allele] = append(cgfi.varid[allele], knot[allele][idx].VarId)
          cgfi.span[allele]  = append(cgfi.span[allele], knot[allele][idx].Span)
          cgfi.step[allele]  = append(cgfi.step[allele], knot[allele][idx].Step)
        }
      }

      patho.loqi.loqi_info[anchor_step] = cgfi


    }
    */

    for anchor_step := range patho.loqi.loqi_info {

      fmt.Printf("anchor_step(%d):\n", anchor_step)
      cgfi := patho.loqi.loqi_info[anchor_step]


      for allele:=0; allele<2; allele++ {

        run_span := 0

        fmt.Printf("  [%d]\n", allele)
        for idx:=0; idx<len(cgfi.step[allele]); idx++ {

          fmt.Printf("    %x(%x).%x+%x:",
            cgfi.step[allele][idx],
            anchor_step + run_span,
            cgfi.varid[allele][idx],
            cgfi.span[allele][idx])

          for i:=0; i<len(cgfi.nocall_start_len[allele][idx]); i+=2 {
            fmt.Printf(" %d+%d",
              cgfi.nocall_start_len[allele][idx][i],
              cgfi.nocall_start_len[allele][idx][i+1])
          }
          fmt.Printf("\n")

          run_span += cgfi.span[allele][idx]

        }
      }

    }

  }

  return PathBytes, nil

}

func _tf(b bool) string {
  if b { return "t" }
  return "."
}
func _tf_(b bool, s string) string {
  if b { return s }
  return "."
}
