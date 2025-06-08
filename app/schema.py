from pydantic import BaseModel


# Type hint for attributes of a house
class HouseInfo(BaseModel):
    MSSubClass: int = 60
    MSZoning: str = "RL"
    LotArea: int = 7844
    LotConfig: str = "Inside"
    BldgType: str = "1Fam"
    OverallCond: int = 7
    YearBuilt: int = 1978
    YearRemodAdd: int = 1978
    Exterior1st: str = "HdBoard"
    BsmtFinSF2: float = 0.0
    TotalBsmtSF: float = 672.0


# Type hint for all the predictions of a house
class HousePrediction(BaseModel):
    Price: float
