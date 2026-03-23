package engine

// Add a (pointer to) variable to the script world
func declVar(name string, value any, doc string) {
	World.Var(name, value, doc)
	if v, ok := value.(Quantity); ok {
		Quantities[name] = v
	}
}

// // Add a (pointer to) variable to the script world
// func declVarWithCallback(name string, value any, doc string, callback func()) {
// 	World.Var(name, value, doc)
// 	addQuantity(name, value, doc)
// 	callback()
// }

// Hack for fixing the closure caveat:
// Defines "t", the time variable, handled specially by Fix()
func declTVar(name string, value any, doc string) {
	World.TVar(name, value, doc)
	addQuantity(name, value, doc)
}

func init() {
	declTVar("t", &Time, "Total simulated time (s)")

	declVar("EnableDemag", &EnableDemag, "Enables/disables demag (default=true)")
	declVar("DemagAccuracy", &DemagAccuracy, "Controls accuracy of demag kernel")
	declVar("DemagBoundaryCorr", &DemagBoundaryCorr, "Enables experimental sparse local demag boundary correction on the boundary shell. Current v1 implementation applies a GPU-side precomputed refined-subgrid tensor correction after the FFT demag field.")
	declVar("DemagBoundaryRadius", &DemagBoundaryRadius, "Neighborhood radius used by the experimental local demag boundary correction stencil.")
	declVar("DemagBoundaryRefine", &DemagBoundaryRefine, "Refinement factor used for local subgrid demag boundary correction precomputation inside boundary-shell cells.")
	declVar("DemagBoundaryHalo", &DemagBoundaryHalo, "Halo size used to dilate the demag boundary shell around partial/cut cells.")
	declVar("DemagBoundaryTol", &DemagBoundaryTol, "Reserved tolerance parameter for future higher-order local demag boundary correction precomputation.")
	declVar("DemagBoundaryPhiFloor", &DemagBoundaryPhiFloor, "Minimum cut-cell volume fraction allowed to participate in local demag boundary correction. Cells below this threshold are excluded to avoid unstable self-demag in pathological sliver cells. The effective floor is max(DemagBoundaryPhiFloor, GeomPhiFloor).")

	declVar("step", &NSteps, "Total number of time steps taken")
	declVar("MinDt", &MinDt, "Minimum time step the solver can take (s)")
	declVar("MaxDt", &MaxDt, "Maximum time step the solver can take (s)")
	declVar("MaxErr", &MaxErr, "Maximum error per step the solver can tolerate (default = 1e-5)")
	declVar("Headroom", &Headroom, "Solver headroom (default = 0.8)")
	declVar("FixDt", &FixDt, "Set a fixed time step, 0 disables fixed step (which is the default)")
	declVar("OpenBC", &OpenBC, "Use open boundary conditions (default=false)")
	declVar("ext_BubbleMz", &BubbleMz, "Center magnetization 1.0 or -1.0  (default = 1.0)")
	declVar("ext_BackGroundTilt", &BackGroundTilt, "Size of in-plane component of background magnetization. All values below this one are rounded down to perfectly out-of-plane to improve position calculation  (default = 0.25)")
	declVar("ext_enableCenterBubbleX", &enableCenterBubbleX, "Enables centering along the X-axis during ext_centerBubble (default=true)")
	declVar("ext_enableCenterBubbleY", &enableCenterBubbleY, "Enables centering along the Y-axis during ext_centerBubble (default=true)")
	declVar("ext_grainCutShape", &grainCutShape, "Whether to add the complete (3D) voronoi grain, only if its centre lies within the shape (default=false)")
	declVar("EdgeSmooth", &edgeSmooth, "Geometry edge smoothing with edgeSmooth^3 samples per cell, 0=staircase, ~8=very smooth")
	declVar("GeomMode", &GeomMode, `Geometry metrics mode: "cutcell" uses voxelizer-derived Phi/Fx/Fy/Fz when available, "legacy" falls back to EdgeSmooth/inside-outside sampling`)
	declVar("GeomTol", &GeomTol, "Adaptive quadrature tolerance for cut-cell waveguide voxelizers")
	declVar("GeomMaxDepth", &GeomMaxDepth, "Maximum adaptive subdivision depth for cut-cell waveguide voxelizers")
	declVar("GeomPhiFloor", &GeomPhiFloor, "Minimum effective cut-cell volume fraction used when normalizing exchange and DMI. Increase this if cut-cell geometries make the solver unstable.")
	declVar("GuideDiagBins", &GuideDiagBins, "Number of arc-length bins used by SaveGuideDiagnostics* when accumulating guide profiles.")
	declVar("GuideDiagSubsample", &GuideDiagSubsample, "Per-axis subsampling for partially filled guide cells in SaveGuideDiagnostics*. A value of 2 means 2^3 test points per cut cell.")
	declVar("GuideDiagCutOnly", &GuideDiagCutOnly, "If true, SaveGuideDiagnostics* samples full cells once at the center and only subsamples partial guide cells.")
	declVar("GuideProjectionEnabled", &GuideProjectionEnabled, "Enables guide-projected cut-cell geometry construction for guide-aware shapes such as SinWaveguideNormal and ArchWaveguideNormal.")
	declVar("GuideProjectionRefine", &GuideProjectionRefine, "Refinement factor used by guide_projection when building the auxiliary Cartesian ROI.")
	declVar("GuideProjectionHalo", &GuideProjectionHalo, "Number of coarse cells of halo added around the guide ROI before fine-grid deposition.")
	declVar("GuideProjectionDS", &GuideProjectionDS, "Guide-projection sampling step along arc length s. Set to 0 to auto-pick the fine-grid scale.")
	declVar("GuideProjectionDV", &GuideProjectionDV, "Guide-projection sampling step across local width v. Set to 0 to auto-pick the fine-grid scale.")
	declVar("GuideProjectionDW", &GuideProjectionDW, "Guide-projection sampling step across local thickness w. Set to 0 to auto-pick the fine-grid scale.")
	declVar("GuideProjectionUseCIC", &GuideProjectionUseCIC, "If true, guide_projection deposits local microvoxels to the fine ROI using CIC instead of nearest-cell deposition.")

	declVar("Tx", &Mesh.Tx, "")
	declVar("Ty", &Mesh.Ty, "")
	declVar("Tz", &Mesh.Tz, "")
	declVar("Nx", &Mesh.Nx, "")
	declVar("Ny", &Mesh.Ny, "")
	declVar("Nz", &Mesh.Nz, "")
	declVar("dx", &Mesh.Dx, "")
	declVar("dy", &Mesh.Dy, "")
	declVar("dz", &Mesh.Dz, "")
	declVar("PBCx", &Mesh.PBCx, "")
	declVar("PBCy", &Mesh.PBCy, "")
	declVar("PBCz", &Mesh.PBCz, "")
	declVar("MinimizerStop", &stopMaxDm, "Stopping max dM for Minimize")
	declVar("MinimizerSamples", &dmSamples, "Number of max dM to collect for Minimize convergence check.")
	declVar("MinimizeMaxSteps", &minimizeMaxSteps, "")
	declVar("MinimizeMaxTimeSeconds", &minimizeMaxTimeSeconds, "")
	declVar("RelaxTorqueThreshold", &relaxTorqueThreshold, "MaxTorque threshold for relax(). If set to -1 (default), relax() will stop when the average torque is steady or increasing.")
	declVar("SnapshotFormat", &snapshotFormat, "Image format for snapshots: jpg, png or gif.")

	declVar("ShiftMagL", &shiftMagL, "Upon shift, insert this magnetization from the left")
	declVar("ShiftMagR", &shiftMagR, "Upon shift, insert this magnetization from the right")
	declVar("ShiftMagU", &shiftMagU, "Upon shift, insert this magnetization from the top")
	declVar("ShiftMagD", &shiftMagD, "Upon shift, insert this magnetization from the bottom")
	declVar("ShiftM", &shiftM, "Whether Shift() acts on magnetization")
	declVar("ShiftGeom", &shiftGeom, "Whether Shift() acts on geometry")
	declVar("ShiftRegions", &shiftRegions, "Whether Shift() acts on regions")
	declVar("TotalShift", &totalShift, "Amount by which the simulation has been shifted (m).")
	declVar("EdgeCarryShift", &EdgeCarryShift, "Whether to use the current magnetization at the border for the cells inserted by Shift")

	declVar("GammaLL", &gammaLL, "Gyromagnetic ratio in rad/Ts")
	declVar("DisableZhangLiTorque", &disableZhangLiTorque, "Disables Zhang-Li torque (default=false)")
	declVar("DisableSlonczewskiTorque", &disableSlonczewskiTorque, "Disables Slonczewski torque (default=false)")
	declVar("DoPrecess", &precess, "Enables LL precession (default=true)")

	declVar("PreviewXDataPoints", &PreviewXDataPoints, "Number of data points in the x direction for the 2D/3D preview")
	declVar("PreviewYDataPoints", &PreviewYDataPoints, "Number of data points in the y direction for the 2D/3D preview")
}

var (
	PreviewXDataPoints = 100
	PreviewYDataPoints = 100
)
