import datetime
import io
import logging
import pprint
import struct

logger = logging.getLogger('int_collector')

class HopMetadata:
    def __init__(self, data, ins_map, int_version=1):
        self.data = data
        self.ins_map = ins_map
        
        self.__parse_switch_id()
        self.__parse_ports()
        self.__parse_hop_latency()
        self.__parse_queue_occupancy()
        
        self.__parse_ingress_timestamp()
        self.__parse_egress_timestamp()
        if int_version == 0:
            self.__parse_queue_congestion()
        elif int_version >= 1:
            self.__parse_l2_ports()
        self.__parse_egress_port_tx_util()
        
    def __parse_switch_id(self):
        if self.ins_map & 0x80:
            self.switch_id = int.from_bytes(self.data.read(4), byteorder='big')
            logger.debug('parse switch id: %d' % self.switch_id)
        
    def __parse_ports(self):
        if self.ins_map & 0x40:
            self.l1_ingress_port_id = int.from_bytes(self.data.read(2), byteorder='big')
            self.l1_egress_port_id = int.from_bytes(self.data.read(2), byteorder='big')
            logger.debug('parse ingress port: %d, egress_port: %d' % (self.l1_ingress_port_id , self.l1_egress_port_id))
        
    def __parse_hop_latency(self):
        if self.ins_map & 0x20:
            self.hop_latency  = int.from_bytes(self.data.read(4), byteorder='big')
            logger.debug('parse hop latency: %d' %  self.hop_latency)
    
    def __parse_queue_occupancy(self):
        if self.ins_map & 0x10:
            self.queue_occupancy_id = int.from_bytes(self.data.read(1), byteorder='big')
            self.queue_occupancy = int.from_bytes(self.data.read(3), byteorder='big')
            logger.debug('parse queue_occupancy_id: %d, queue_occupancy: %d' % (self.queue_occupancy_id, self.queue_occupancy))
            
    def __parse_ingress_timestamp(self):
        if self.ins_map & 0x08:
            self.ingress_timestamp  = int.from_bytes(self.data.read(8), byteorder='big')
            logger.debug('parse ingress_timestamp: %d' %  self.ingress_timestamp)
            
    def __parse_egress_timestamp(self):
        if self.ins_map & 0x04:
            self.egress_timestamp  = int.from_bytes(self.data.read(8), byteorder='big')
            logger.debug('parse egress_timestamp: %d' %  self.egress_timestamp)
     
    def  __parse_queue_congestion(self):
        if self.ins_map & 0x02:
            self.queue_congestion_id = int.from_bytes(self.data.read(1), byteorder='big')
            self.queue_congestion = int.from_bytes(self.data.read(3), byteorder='big')
            logger.debug('parse queue_congestion_id: %d, queue_congestion: %d' % (self.queue_congestion_id, self.queue_congestion))
            
    def  __parse_l2_ports(self):
        if self.ins_map & 0x02:
            self.l2_ingress_port_id = int.from_bytes(self.data.read(2), byteorder='big')
            self.l2_egress_port_id = int.from_bytes(self.data.read(2), byteorder='big')
            logger.debug('parse L2 ingress port: %d, egress_port: %d' % (self.l2_ingress_port_id , self.l2_egress_port_id))
            
    def  __parse_egress_port_tx_util(self):
        if self.ins_map & 0x01:
            self.egress_port_tx_util = int.from_bytes(self.data.read(4), byteorder='big')
            logger.debug('parse egress_port_tx_util: %d' % self.egress_port_tx_util)
            
    def unread_data(self):
        return self.data
            
    def __str__(self):
        attrs = vars(self)
        try:
            del attrs['data']
            del attrs['ins_map']
        except Exception as e: 
            logger.error(e)
        return pprint.pformat(attrs)

def ip2str(ip):
    return "{}.{}.{}.{}".format(ip[0],ip[1],ip[2],ip[3])

# ethernet(14B) + IP(20B) + UDP(8B)
UDP_OFFSET = 14 + 20 + 8
# ethernet(14B) + IP(20B) + TCP(20B)
TCP_OFFSET = 14 + 20 + 20

class IntReport():
    def __init__(self, data):
        orig_data = data
        #data = struct.unpack("!%dB" % len(data), data)
        '''
        header int_report_fixed_header_t {
            bit<4> ver;
            bit<4> len;
            bit<3> nprot;
            bit<6> rep_md_bits;
            bit<6> reserved;
            bit<1> d;
            bit<1> q;
            bit<1> f;
            bit<6> hw_id;
            bit<32> switch_id;
            bit<32> seq_num;
            bit<32> ingress_tstamp;
        }
        const bit<8> REPORT_FIXED_HEADER_LEN = 16;
        '''
        
        # report header
        self.int_report_hdr = data[:16]
        self.ver = self.int_report_hdr[0] >> 4
        self.rcvd_at = datetime.datetime.now()
        
        if self.ver != 1:
            logger.error("Unsupported INT report version %s - skipping report" % self.int_version)
            raise Exception("Unsupported INT report version %s - skipping report" % self.int_version)
        
        self.len = self.int_report_hdr[0] & 0x0f
        self.nprot = self.int_report_hdr[1] >> 5
        self.rep_md_bits = (self.int_report_hdr[1] & 0x1f) + (self.int_report_hdr[2] >> 7)
        self.d = self.int_report_hdr[2] & 0x01
        self.q = self.int_report_hdr[3] >> 7
        self.f = (self.int_report_hdr[3] >> 6) & 0x01
        self.hw_id = self.int_report_hdr[3] & 0x3f
        self.switch_id, self.seq_num, self.ingress_tstamp = struct.unpack('!3I', orig_data[4:16])

        # flow id
        self.ip_hdr = data[30:50]
        self.udp_hdr = data[50:58]
        protocol = self.ip_hdr[9]
        self.flow_id = {
            'srcip': ip2str(self.ip_hdr[12:16]),
            'dstip': ip2str(self.ip_hdr[16:20]), 
            'srcp': struct.unpack('!H', self.udp_hdr[:2])[0],
            'dstp': struct.unpack('!H', self.udp_hdr[2:4])[0],
            'protocol': self.ip_hdr[9],       
        }

        # check next protocol
        # offset: udp/tcp + report header(16B)
        offset = 16
        if protocol == 17:
            offset = offset + UDP_OFFSET
        if protocol == 6:
            offset = offset + TCP_OFFSET

        '''
        header intl4_shim_t {
            bit<8> int_type;
            bit<8> rsvd1;
            bit<8> len;   // the length of all INT headers in 4-byte words
            bit<6> rsvd2;  // dscp not put here
            bit<2> rsvd3;
        }
        const bit<16> INT_SHIM_HEADER_LEN_BYTES = 4;
        '''
        # int shim
        self.int_shim = data[offset:offset + 4]
        self.int_type = self.int_shim[0]
        self.int_data_len = int(self.int_shim[2]) - 3
        
        if self.int_type != 1: 
            logger.error("Unsupported INT type %s - skipping report" % self.int_type)
            raise Exception("Unsupported INT type %s - skipping report" % self.int_type)
  
        '''  INT header version 0.4     
        header int_header_t {
            bit<2> ver;
            bit<2> rep;
            bit<1> c;
            bit<1> e;
            bit<5> rsvd1;
            bit<5> ins_cnt;  // the number of instructions that are set in the instruction mask
            bit<8> max_hops; // maximum number of hops inserting INT metadata
            bit<8> total_hops; // number of hops that inserted INT metadata
            bit<16> instruction_mask;
            bit<16> rsvd2;
        }'''
        
        '''  INT header version 1.0
        header int_header_t {
            bit<4>  ver;
            bit<2>  rep;
            bit<1>  c;
            bit<1>  e;
            bit<1>  m;
            bit<7>  collection_id;
            bit<3>  rsvd2;
            bit<5>  hop_metadata_len;   // the length of the metadata added by a single INT node (4-byte words)
            bit<8>  remaining_hop_cnt;  // how many switches can still add INT metadata
            bit<16>  instruction_mask;   
            bit<16> rsvd3;
        }'''


        # int header
        self.int_hdr = data[offset + 4:offset + 12]
        self.int_version = self.int_hdr[0] >> 4  # version in INT v0.4 has only 2 bits!
        if self.int_version == 0: # if rep is 0 then it is ok for INT v0.4
            self.hop_count = self.int_hdr[3]
        elif self.int_version == 1:
            self.collection_id = int(self.int_hdr[1] & 0x7f)
            self.hop_metadata_len = int(self.int_hdr[2] & 0x1f)
            self.remaining_hop_cnt = self.int_hdr[3]
            self.hop_count = int(self.int_data_len / self.hop_metadata_len)

            logger.debug("hop_metadata_len: %d, int_data_len: %d, remaining_hop_cnt: %d, hop_count: %d" % (
                            self.hop_metadata_len, self.int_data_len, self.remaining_hop_cnt, self.hop_count))
        else:
            logger.error("Unsupported INT version %s - skipping report" % self.int_version)
            raise Exception("Unsupported INT version %s - skipping report" % self.int_version)

        self.ins_map = int.from_bytes(self.int_hdr[4:6], byteorder='big')
        first_slice = (self.ins_map & 0b0000111100000000) << 4
        second_slice = (self.ins_map & 0b1111000000000000) >> 4
        self.ins_map = (first_slice + second_slice) >> 8
        

        # int metadata
        self.int_meta = data[offset + 12:]
        self.hop_metadata: list[HopMetadata] = []
        self.int_meta = io.BytesIO(self.int_meta)
        for i in range(self.hop_count):
            try:
                hop = HopMetadata(self.int_meta, self.ins_map, self.int_version)
                self.int_meta = hop.unread_data()
                self.hop_metadata.append(hop)
            except Exception as e:
                logger.info("Metadata left (%s position) is: %s" % (self.int_meta.tell(), self.int_meta))
                logger.error(e)
                break
        
        # newer bmv2 (/p4c) version breaks propagating ingress metadata in cloning and we get invalid
        # ingress timestamp, this fixes it by assigning egress timestamp of sink's predecessor to it's ingress tstamp
        self.hop_metadata.reverse()
        if len(self.hop_metadata) > 1:
            pre_sink_egress_time = self.hop_metadata[-2].egress_timestamp
            self.hop_metadata[-1].ingress_timestamp = pre_sink_egress_time
            self.hop_metadata[-1].hop_latency = int((self.hop_metadata[-1].egress_timestamp - pre_sink_egress_time) / 1000.0)

        logger.debug(vars(self))
            
    def __str__(self):
        hop_info = ''
        for hop in self.hop_metadata:
            hop_info += str(hop) + '\n'
        flow_tuple = "src_ip: %(srcip)s, dst_ip: %(dstip)s, src_port: %(srcp)s, dst_port: %(dstp)s, protocol: %(protocol)s" % self.flow_id 
        additional_info =  "sw: %s, seq: %s, int version: %s, ins_map: 0x%x, hops: %d" % (
            self.switch_id,
            self.seq_num,
            self.int_version,
            self.ins_map,
            self.hop_count,
        )
        return "\n".join([flow_tuple, additional_info, hop_info])
        
