package shimutils

type SdnShimService struct {

}

func NewSdnShimService() *SdnShimService {
	return &SdnShimService{}
}

func (s *SdnShimService) AssertAreTransit(switchNames []string) {

}

func (s *SdnShimService) AssertIsSink(switchName string, neighborName string, collectorIpv4 string) {

}

func (s *SdnShimService) AssertIsSource(switchName string, sourceNeighborName string, sourceConfig interface{}) {

}
